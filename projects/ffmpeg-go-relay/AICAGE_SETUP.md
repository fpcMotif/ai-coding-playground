# AICage (Droid Agent AI) Setup & Research

## Research: What is "Droid Agent AI"?
"Droid agent ai" appears to refer to **aicage** (`aicage/aicage` on GitHub).
- **Purpose**: Runs agentic coding assistants (Codex, Copilot, Gemini, etc.) inside Docker containers for security and isolation.
- **Key Feature**: "Cage" your agents so they don't have unrestricted access to your host machine.
- **Support**: Supports OpenAI Codex via `aicage codex`.

## Configuration Status
I have performed the following configuration steps for you:

1.  **Installed Prerequisites**:
    - Installed `pipx` (using Homebrew).
    - Added `pipx` to PATH.

2.  **Installed aicage**:
    - Ran `pipx install aicage`.
    - Version installed: `0.8.17`.

3.  **Configured Project**:
    - Created a project-specific configuration for `/Users/f/ffmpeg-go-relay`.
    - Config file: `~/.aicage/projects/75be9dde678cbe7db7f80461ad1b563e414dc1ad4db90177c62157d4de6245aa.yaml`
    - Settings:
        - Agent: `codex`
        - Base Image: `ubuntu`

## How to Run
To use Codex in "droid" (aicage), run:

```bash
aicage codex
```

## Important Note
**Docker Daemon Requirement**:
The `aicage` tool requires a running Docker daemon. I detected that the Docker daemon is currently **not running** or accessible (`/var/run/docker.sock` is missing).
Please start Docker Desktop or the Docker daemon on your machine before running `aicage`.
