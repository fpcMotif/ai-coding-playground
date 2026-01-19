package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"
)

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
	ID      json.RawMessage `json:"id,omitempty"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

type callParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

type feedbackArgs struct {
	Message       string   `json:"message"`
	Options       []string `json:"options,omitempty"`
	AllowFreeText *bool    `json:"allowFreeText,omitempty"`
	DefaultOption string   `json:"defaultOption,omitempty"`
	TimeoutSec    *int     `json:"timeoutSec,omitempty"`
}

func main() {
	decoder := json.NewDecoder(os.Stdin)
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetEscapeHTML(false)

	for {
		var req rpcRequest
		if err := decoder.Decode(&req); err != nil {
			if errors.Is(err, io.EOF) {
				return
			}
			fmt.Fprintln(os.Stderr, "decode error:", err)
			continue
		}

		if req.Method == "exit" && len(req.ID) == 0 {
			return
		}

		switch req.Method {
		case "initialize":
			result := map[string]any{
				"protocolVersion": "2024-11-05",
				"serverInfo": map[string]any{
					"name":    "interactive-feedback-mcp",
					"version": "0.1.0",
				},
				"capabilities": map[string]any{
					"tools": map[string]any{
						"listChanged": false,
					},
				},
			}
			writeResult(encoder, req.ID, result)
		case "tools/list":
			result := map[string]any{
				"tools": []map[string]any{
					{
						"name":        "interactive_feedback",
						"description": "Prompt the user and return their response.",
						"inputSchema": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"message": map[string]any{
									"type":        "string",
									"description": "Prompt shown to the user.",
								},
								"options": map[string]any{
									"type":        "array",
									"items":       map[string]any{"type": "string"},
									"description": "Optional list of choices.",
								},
								"allowFreeText": map[string]any{
									"type":        "boolean",
									"description": "Allow free text input when options are provided.",
								},
								"defaultOption": map[string]any{
									"type":        "string",
									"description": "Default option when user submits empty input.",
								},
								"timeoutSec": map[string]any{
									"type":        "integer",
									"minimum":     0,
									"description": "Timeout in seconds (0 disables timeout).",
								},
							},
							"required": []string{"message"},
						},
					},
				},
			}
			writeResult(encoder, req.ID, result)
		case "tools/call":
			result := handleToolCall(req.Params)
			writeResult(encoder, req.ID, result)
		case "shutdown":
			writeResult(encoder, req.ID, map[string]any{})
		default:
			writeError(encoder, req.ID, -32601, "method not found")
		}
	}
}

func handleToolCall(params json.RawMessage) map[string]any {
	var call callParams
	if err := json.Unmarshal(params, &call); err != nil {
		return toolError("invalid params")
	}

	if call.Name != "interactive_feedback" {
		return toolError("unknown tool")
	}

	var args feedbackArgs
	if len(call.Arguments) > 0 {
		if err := json.Unmarshal(call.Arguments, &args); err != nil {
			return toolError("invalid arguments")
		}
	}

	answer, err := promptFeedback(args)
	if err != nil {
		return toolError(err.Error())
	}

	return map[string]any{
		"content": []map[string]any{
			{
				"type": "text",
				"text": answer,
			},
		},
		"isError": false,
	}
}

func promptFeedback(args feedbackArgs) (string, error) {
	if strings.TrimSpace(args.Message) == "" {
		return "", errors.New("message is required")
	}

	allowFreeText := true
	if len(args.Options) > 0 {
		allowFreeText = false
	}
	if args.AllowFreeText != nil {
		allowFreeText = *args.AllowFreeText
	}

	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return "", errors.New("/dev/tty unavailable")
	}
	defer tty.Close()

	reader := bufio.NewReader(tty)
	fmt.Fprintln(tty, args.Message)
	if len(args.Options) > 0 {
		for i, opt := range args.Options {
			fmt.Fprintf(tty, "%d) %s\n", i+1, opt)
		}
	}
	if args.DefaultOption != "" {
		fmt.Fprintf(tty, "Default: %s\n", args.DefaultOption)
	}
	fmt.Fprint(tty, "> ")

	inputCh := make(chan string, 1)
	errCh := make(chan error, 1)
	go func() {
		line, readErr := reader.ReadString('\n')
		if readErr != nil && !errors.Is(readErr, io.EOF) {
			errCh <- readErr
			return
		}
		inputCh <- strings.TrimSpace(line)
	}()

	var input string
	if args.TimeoutSec != nil && *args.TimeoutSec > 0 {
		select {
		case input = <-inputCh:
		case err = <-errCh:
			return "", err
		case <-time.After(time.Duration(*args.TimeoutSec) * time.Second):
			if args.DefaultOption != "" {
				return args.DefaultOption, nil
			}
			return "", errors.New("timeout waiting for input")
		}
	} else {
		select {
		case input = <-inputCh:
		case err = <-errCh:
			return "", err
		}
	}

	if input == "" && args.DefaultOption != "" {
		return args.DefaultOption, nil
	}
	if len(args.Options) == 0 {
		if strings.TrimSpace(input) == "" {
			return "", errors.New("empty input")
		}
		return input, nil
	}

	if idx, convErr := strconv.Atoi(input); convErr == nil {
		if idx >= 1 && idx <= len(args.Options) {
			return args.Options[idx-1], nil
		}
	}

	for _, opt := range args.Options {
		if input == opt {
			return opt, nil
		}
		if strings.EqualFold(input, opt) {
			return opt, nil
		}
	}

	if allowFreeText {
		if strings.TrimSpace(input) == "" {
			return "", errors.New("empty input")
		}
		return input, nil
	}

	return "", errors.New("invalid selection")
}

func toolError(message string) map[string]any {
	return map[string]any{
		"content": []map[string]any{
			{
				"type": "text",
				"text": message,
			},
		},
		"isError": true,
	}
}

func writeResult(encoder *json.Encoder, id json.RawMessage, result any) {
	if len(id) == 0 {
		return
	}
	resp := rpcResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
	if err := encoder.Encode(resp); err != nil {
		fmt.Fprintln(os.Stderr, "encode error:", err)
	}
}

func writeError(encoder *json.Encoder, id json.RawMessage, code int, message string) {
	if len(id) == 0 {
		return
	}
	resp := rpcResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &rpcError{
			Code:    code,
			Message: message,
		},
	}
	if err := encoder.Encode(resp); err != nil {
		fmt.Fprintln(os.Stderr, "encode error:", err)
	}
}
