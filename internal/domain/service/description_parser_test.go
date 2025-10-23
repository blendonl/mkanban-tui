package service

import (
	"testing"
)

func TestParseSubtasks(t *testing.T) {
	tests := []struct {
		name        string
		description string
		expected    []string
	}{
		{
			name: "single subtask",
			description: `This is a task description.

- [ ] Implement feature A

Some more text.`,
			expected: []string{"Implement feature A"},
		},
		{
			name: "multiple subtasks",
			description: `Task with multiple subtasks:

- [ ] First task
- [ ] Second task
- [ ] Third task`,
			expected: []string{"First task", "Second task", "Third task"},
		},
		{
			name: "mixed checkboxes",
			description: `Mixed checkboxes:

- [ ] Unchecked task
- [x] Already done task
- [~] In progress task`,
			expected: []string{"Unchecked task"},
		},
		{
			name: "no subtasks",
			description: `Just a regular description.

No checkboxes here!`,
			expected: []string{},
		},
		{
			name: "linked checkboxes ignored",
			description: `Already processed:

- [ ] [Task Link](../../Todo/BOARD-1/task.md)`,
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseSubtasks(tt.description)
			if len(result) != len(tt.expected) {
				t.Errorf("expected %d subtasks, got %d", len(tt.expected), len(result))
				return
			}
			for i, expected := range tt.expected {
				if result[i] != expected {
					t.Errorf("subtask %d: expected %q, got %q", i, expected, result[i])
				}
			}
		})
	}
}

func TestUpdateCheckboxState(t *testing.T) {
	description := `Task description:

- [ ] [First task](../../Todo/BOARD-2-first-task/task.md)
- [ ] [Second task](../../Todo/BOARD-3-second-task/task.md)`

	// Update first task to done
	updated := UpdateCheckboxState(description, "BOARD-2-first-task", CheckboxDone)
	expected := `Task description:

- [x] [First task](../../Todo/BOARD-2-first-task/task.md)
- [ ] [Second task](../../Todo/BOARD-3-second-task/task.md)`

	if updated != expected {
		t.Errorf("expected:\n%s\n\ngot:\n%s", expected, updated)
	}

	// Update second task to in progress
	updated = UpdateCheckboxState(updated, "BOARD-3-second-task", CheckboxInProgress)
	expected = `Task description:

- [x] [First task](../../Todo/BOARD-2-first-task/task.md)
- [~] [Second task](../../Todo/BOARD-3-second-task/task.md)`

	if updated != expected {
		t.Errorf("expected:\n%s\n\ngot:\n%s", expected, updated)
	}
}

func TestGetCheckboxStates(t *testing.T) {
	description := `Task description:

- [x] [First task](../../Done/BOARD-2-first-task/task.md)
- [~] [Second task](../../In Progress/BOARD-3-second-task/task.md)
- [ ] [Third task](../../Todo/BOARD-4-third-task/task.md)`

	states := GetCheckboxStates(description)

	if len(states) != 3 {
		t.Errorf("expected 3 states, got %d", len(states))
	}

	if states["BOARD-2-first-task"] != CheckboxDone {
		t.Errorf("expected first task to be done, got %v", states["BOARD-2-first-task"])
	}

	if states["BOARD-3-second-task"] != CheckboxInProgress {
		t.Errorf("expected second task to be in progress, got %v", states["BOARD-3-second-task"])
	}

	if states["BOARD-4-third-task"] != CheckboxTodo {
		t.Errorf("expected third task to be todo, got %v", states["BOARD-4-third-task"])
	}
}

func TestAllCheckboxesComplete(t *testing.T) {
	tests := []struct {
		name        string
		description string
		expected    bool
	}{
		{
			name: "all complete",
			description: `Task:

- [x] [Task 1](../../Done/BOARD-2/task.md)
- [x] [Task 2](../../Done/BOARD-3/task.md)`,
			expected: true,
		},
		{
			name: "some incomplete",
			description: `Task:

- [x] [Task 1](../../Done/BOARD-2/task.md)
- [ ] [Task 2](../../Todo/BOARD-3/task.md)`,
			expected: false,
		},
		{
			name: "no checkboxes",
			description: `Just a regular task.`,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := AllCheckboxesComplete(tt.description)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestAddSubtaskLink(t *testing.T) {
	description := `Task description:

- [ ] Implement feature A
- [ ] Add tests`

	// Add link to first subtask
	updated := AddSubtaskLink(description, "Implement feature A", "BOARD-2-implement-feature-a", "Todo")
	expected := `Task description:

- [ ] [Implement feature A](../../Todo/BOARD-2-implement-feature-a/task.md)
- [ ] Add tests`

	if updated != expected {
		t.Errorf("expected:\n%s\n\ngot:\n%s", expected, updated)
	}

	// Add link to second subtask
	updated = AddSubtaskLink(updated, "Add tests", "BOARD-3-add-tests", "Todo")
	expected = `Task description:

- [ ] [Implement feature A](../../Todo/BOARD-2-implement-feature-a/task.md)
- [ ] [Add tests](../../Todo/BOARD-3-add-tests/task.md)`

	if updated != expected {
		t.Errorf("expected:\n%s\n\ngot:\n%s", expected, updated)
	}
}
