#!/bin/bash

# sisyphus-wrapper.sh
# Wrapper for Sisyphus (Oh My OpenCode) to run in Tmux

echo "=================================================="
echo "      SISYPHUS - The Task Enforcer Agent          "
echo "=================================================="
echo "Current Task Context:"
# Optionally show git status or recent changes
git status --short 2>/dev/null

echo ""
echo "Enter the task you want Sisyphus to complete:"
read -p "> " task

if [ -n "$task" ]; then
    echo "Starting Sisyphus with task: $task"
    oh-my-opencode run "$task"
else
    echo "No task provided. Exiting."
fi

# Keep window open if it crashes
echo "Press Enter to close..."
read
