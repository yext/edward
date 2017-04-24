package tracker

import (
	"os"
	"reflect"
	"testing"
)

func TestTracker(t *testing.T) {
	var tests = []struct {
		name string
		jobs []struct {
			name    string
			state   TaskState
			message []string
		}
		expectedMessage []string
		expectedState   TaskState
	}{
		{
			name:          "empty",
			expectedState: TaskStatePending,
		},
		{
			name: "Single job, in progress",
			jobs: []struct {
				name    string
				state   TaskState
				message []string
			}{
				{name: "job1", message: []string{"testState"}},
			},
			expectedState: TaskStatePending,
		},
		{
			name: "Single job, succeeded",
			jobs: []struct {
				name    string
				state   TaskState
				message []string
			}{
				{name: "job1", message: []string{"testState"}, state: TaskStateSuccess},
			},
			expectedState: TaskStateSuccess,
		},
		{
			name: "All states",
			jobs: []struct {
				name    string
				state   TaskState
				message []string
			}{
				{name: "job1"},
				{name: "job2", state: TaskStateSuccess},
				{name: "job3", state: TaskStateWarning},
				{name: "job4", state: TaskStateFailed},
			},
			expectedState: TaskStateFailed,
		},
		{
			name: "Some finished, some pending",
			jobs: []struct {
				name    string
				state   TaskState
				message []string
			}{
				{name: "job1"},
				{name: "job2", state: TaskStateSuccess},
				{name: "job3", state: TaskStateWarning},
			},
			expectedState: TaskStateInProgress,
		},
		{
			name: "Some finished, some in progress",
			jobs: []struct {
				name    string
				state   TaskState
				message []string
			}{
				{name: "job1", state: TaskStateInProgress},
				{name: "job2", state: TaskStateSuccess},
				{name: "job3", state: TaskStateWarning},
			},
			expectedState: TaskStateInProgress,
		},
		{
			name: "Finished with warning",
			jobs: []struct {
				name    string
				state   TaskState
				message []string
			}{
				{name: "job2", state: TaskStateSuccess},
				{name: "job3", state: TaskStateWarning},
			},
			expectedState: TaskStateWarning,
		},
		{
			name: "Failure stops progress",
			jobs: []struct {
				name    string
				state   TaskState
				message []string
			}{
				{name: "job1"},
				{name: "very_long_job_name", state: TaskStateFailed},
				{name: "job2", state: TaskStateSuccess},
				{name: "job3", state: TaskStateWarning},
			},
			expectedState: TaskStateFailed,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			task := NewTask()
			if task.Name() != "" {
				t.Errorf("Task name was not as expected. Got '%v'", task.Name())
			}
			for _, state := range test.jobs {
				child := task.Child(state.name)
				expectUpdate(t, task)
				child.SetState(state.state, state.message...)
				expectUpdate(t, task)
			}
			orderedChildren := task.Children()
			for index, state := range test.jobs {
				child := task.Child(state.name)
				if orderedChildren[index] != child {
					t.Errorf("Child as index %v was not the expected child", index)
				}
				if child.State() != state.state {
					t.Errorf("Child state for '%v' was '%v', expected '%v'.", state.name, child.State(), state.state)
				}
			}
			result := task.State()
			if result != test.expectedState {
				t.Errorf("State was not as expected. Expected: %v, Got: %v", test.expectedState, result)
			}
			if !reflect.DeepEqual(task.Messages(), test.expectedMessage) {
				t.Errorf("Messages were not as expected")
			}
			renderer := &ANSIRenderer{}
			err := renderer.Render(os.Stdout, task)
			if err != nil {
				t.Errorf(err.Error())
			}
			task.Close()
			expectUpdate(t, task)
		})
	}
}

func expectUpdate(t *testing.T, task Task) {
	select {
	case <-task.Updates():
	default:
		t.Error("Expected state update message")
	}
}
