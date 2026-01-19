#!/bin/bash

# start-dev-session.sh
# Script to start a tmux session with pi and aicage agents ready

SESSION="dev-environment"

# Check if session exists
tmux has-session -t $SESSION 2>/dev/null

if [ $? != 0 ]; then
  # Create new session
  tmux new-session -d -s $SESSION -n "Main"

  # Pane 1: Main shell (User)
  # (Already created by new-session)

  # Pane 2: aicage opencode (OpenCode Agent)
  tmux split-window -h -t $SESSION:1
  tmux send-keys -t $SESSION:1.2 "aicage opencode" C-m

  # Pane 3: aicage codex (Codex Agent)
  tmux split-window -v -t $SESSION:1.1
  tmux send-keys -t $SESSION:1.3 "aicage codex" C-m

  # Layout: Main on left, agents stacked on right
  tmux select-layout -t $SESSION:1 main-vertical

  # Select main pane
  tmux select-pane -t $SESSION:1.1
fi

# Attach to session
tmux attach -t $SESSION
