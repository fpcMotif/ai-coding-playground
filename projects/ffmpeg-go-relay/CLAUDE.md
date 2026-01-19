# Package Management Rules
- **ALWAYS use `bun install`**. NEVER use `npm install`.
- **ALWAYS use `bunx`** instead of `npx`.
- If `bun` fails, **FIX THE ERROR**. DO NOT fallback to `npm`.

# Project Configuration

## Mixedbread mgrep Configuration

- **API Key**: `mxb_1vh2AkkRng5OnyOew0bHJxFbyfRn`
- **Token Location**: `~/.mgrep/token.json`
- **Store Name**: `mgrep` (default)

### mgrep Commands

```bash
# Semantic search in current directory
mgrep "your query" .

# Search with web results and answer
mgrep --web --answer "query"

# Watch for file changes (keeps index updated)
mgrep watch

# Sync files before search
mgrep -s "query"
```

### Claude Code Plugin

mgrep is installed as a Claude Code plugin:
- Plugin location: `~/.claude/plugins/cache/Mixedbread-Grep/mgrep/`
- Settings: `~/.claude/settings.json` has `"mgrep@Mixedbread-Grep": true`

### Integration Notes

- Use `mgrep` instead of `grep` or `rg` for semantic code search
- Use `mgrep --web --answer` instead of web search for questions
- Files are automatically indexed when `mgrep watch` runs on session start

# Modern Tooling & Agent Awareness

This environment uses modern Rust-based alternatives to standard UNIX tools.
**Agents must be aware of the following behavior differences:**

1.  **`ls` is aliased to `eza`**:
    *   Output may contain icons/colors. Use `--oneline` for parsing.
    *   Tree view: `eza -T`.
2.  **`find` is aliased to `fd`**:
    *   **Ignores hidden files & .gitignore by default.**
    *   Use `-H` (hidden) and `-I` (no-ignore) to find all files.
    *   Syntax is `fd pattern` (regex/glob), not `find -name`.
3.  **`grep` is aliased to `rg` (ripgrep)**:
    *   **Ignores .gitignore by default.**
    *   Use `-u` (no-ignore) or `-uu` (hidden + no-ignore) to search everything.
4.  **`cd` is aliased to `zoxide` (z)**:
    *   `z` jumps based on history/frequency.
    *   For deterministic navigation in scripts, use relative/absolute paths.
5.  **`cat` is aliased to `bat`**:
    *   Output has line numbers and grid by default.
    *   Use `--style=plain` for raw output (handled in alias).
6.  **`lazygit` is available**:
    *   Located at `/opt/homebrew/bin/lazygit`.
    *   Aliased to `lazygit`.
    *   **Agent Note:** This is an interactive TUI. Do not run it in headless mode. Use standard `git` commands for operations, or `lazygit` only if explicitly requested for an interactive session (which this agent cannot drive).

**Recommended Usage:**
- Search files: `fd -e go` (find all go files)
- Search text: `rg "func main"`
- List tree: `eza -T -L 2`
