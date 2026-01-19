# Oh My OpenCode & Sisyphus Setup

## Research: What are OMO and Sisyphus?
- **Oh My OpenCode (OMO)**: A powerful "Agent Harness" (`code-yeongyu/oh-my-opencode`) that orchestrates coding agents.
- **Sisyphus**: The core agent within OMO designed to "code like you" and enforce task completion loops.

## Integration Status
I have installed **Oh My OpenCode** (v2.14.0) and configured a "Best Practice" Tmux environment for you.

### 1. New Tools Installed
- `oh-my-opencode` (Global npm package)
- `sisyphus-wrapper.sh` (Helper script for task loops)
- `start-ultimate-dev.sh` (The master startup script)

### 2. The "Ultimate" Dev Environment
Run `./start-ultimate-dev.sh` to launch a `tmux` session with 3 windows:

1.  **Pi-Agent (Window 1)**
    - **Left**: Runs `pi` (Active Coding Agent).
    - **Right**: Shell for manual commands.

2.  **Sisyphus (Window 2)**
    - Runs `omo run` (Sisyphus) in a wrapper.
    - Asks for a task and continuously works on it until completion.

3.  **Droid-Lab (Window 3)**
    - **Top**: `aicage codex` (Deep research/snippets in Docker).
    - **Bottom**: `aicage opencode` (Alternative agent in Docker).

### 3. Next Steps
1.  **Fix Configuration**: OMO reported some configuration errors. Run this to fix them interactively:
    ```bash
    oh-my-opencode install
    ```
2.  **Start Coding**:
    ```bash
    ./start-ultimate-dev.sh
    ```

### Why this is "Best Practice"
- **Separation of Concerns**: Main agent (Pi) separate from Task Enforcer (Sisyphus) separate from Sandboxed Agents (Droid/AICage).
- **Isolation**: Risky/Deep research happens in Droid (Docker).
- **Persistence**: `tmux` keeps your agents running even if you disconnect.
