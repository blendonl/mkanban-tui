#!/bin/bash
# Script to add multiple tasks for testing scrolling functionality

echo "Adding test tasks to mkanban..."

# Add 15 tasks to test scrolling
for i in {1..15}; do
    echo "Task $i: Testing scrolling feature" | \
    ./mkanban new todo >/dev/null 2>&1 || echo "Note: Use the TUI to add tasks manually"
done

echo "Test tasks added! Run ./mkanban to see scrolling in action."
echo "Use arrow keys or j/k to scroll through tasks."
