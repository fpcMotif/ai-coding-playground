#!/bin/bash

# start-ultimate-dev.sh
# The "Sisyphus" configuration for macOS M1
# Integrates Pi, Oh My OpenCode (Sisyphus), and AICage (Codex) in Tmux

SESSION="sisyphus-dev"

# Ensure tools are in PATH
export PATH=$PATH:/opt/homebrew/bin:/Users/f/.local/bin

# Check if session exists
if tmux has-session -t $SESSION 2>/dev/null; then
    echo "Session $SESSION already exists. Attaching..."
    tmux attach -t $SESSION
    exit 0
fi

# Create new session
tmux new-session -d -s $SESSION -n "Control"

# ==================================================
# Window 1: Control Center (Pi & Main Shell)
# ==================================================
# Pane 1 (Left): Pi Agent - Your primary pair programmer
tmux rename-window -t $SESSION:1 "Pi-Agent"
tmux send-keys -t $SESSION:1 "pi" C-m

# Pane 2 (Right): Interactive Shell
tmux split-window -h -t $SESSION:1 -p 40
tmux send-keys -t $SESSION:1.2 "ls -la" C-m

# ==================================================
# Window 2: Sisyphus (Task Enforcer)
# ==================================================
tmux new-window -t $SESSION -n "Sisyphus"
tmux send-keys -t $SESSION:2 "./sisyphus-wrapper.sh" C-m

# ==================================================
# Window 3: Droid Lab (AICage Agents)
# ==================================================
tmux new-window -t $SESSION -n "Droid-Lab"
# Pane 1: Codex (for deep research/snippets)
tmux send-keys -t $SESSION:3 "aicage codex" C-m
# Pane 2: OpenCode (Containerized backup)
tmux split-window -v -t $SESSION:3
tmux send-keys -t $SESSION:3.2 "aicage opencode" C-m

# Select the first window
tmux select-window -t $SESSION:1

# Attach
tmux attach -t $SESSION
