package tracker

import "testing"

func TestInitialState(t *testing.T) {
	operation := NewOperation()
	rendered := operation.RenderState()
	if len(rendered) != 0 {
		t.Errorf("Expected empty render, got: %v\n", string(rendered))
	}
}

func TestOperation(t *testing.T) {
	var tests = []struct {
		name string
		jobs []struct {
			name    string
			state   jobState
			message string
			extra   []string
		}
		expectedOutput string
		expectedDone   bool
	}{
		{
			name:         "empty",
			expectedDone: true,
		},
		{
			name: "Single job, in progress",
			jobs: []struct {
				name    string
				state   jobState
				message string
				extra   []string
			}{
				{name: "job1", message: "testState"},
			},
			expectedOutput: "job1: [testState]",
		},
		{
			name: "All states",
			jobs: []struct {
				name    string
				state   jobState
				message string
				extra   []string
			}{
				{name: "job1", message: "i"},
				{name: "job2", message: "s", state: jobStateSuccess},
				{name: "job3", message: "w", state: jobStateWarning},
				{name: "job4", message: "f", state: jobStateFailed},
			},
			expectedOutput: "job1: [i]\njob2: [s]\njob3: [w]\njob4: [f]",
			expectedDone:   true,
		},
		{
			name: "Some finished, in progress",
			jobs: []struct {
				name    string
				state   jobState
				message string
				extra   []string
			}{
				{name: "job1", message: "i"},
				{name: "job2", message: "s", state: jobStateSuccess},
				{name: "job3", message: "w", state: jobStateWarning},
			},
			expectedOutput: "job1: [i]\njob2: [s]\njob3: [w]",
		},
		{
			name: "Failure stops progress",
			jobs: []struct {
				name    string
				state   jobState
				message string
				extra   []string
			}{
				{name: "job1", message: "i"},
				{name: "job4", message: "f", state: jobStateFailed},
				{name: "job2", message: "s", state: jobStateSuccess},
				{name: "job3", message: "w", state: jobStateWarning},
			},
			expectedOutput: "job1: [i]\njob4: [f]",
			expectedDone:   true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			operation := NewOperation()
			for _, state := range test.jobs {
				job := operation.GetJob(state.name)
				job.(*simpleJob).testRender = true
				if state.state == jobStateInProgress {
					job.State(state.message)
				}
				if state.state == jobStateSuccess {
					job.Success(state.message)
				}
				if state.state == jobStateFailed {
					job.Fail(state.message, state.extra...)
				}
				if state.state == jobStateWarning {
					job.Warning(state.message)
				}
				select {
				case <-operation.StateUpdate():
				default:
					t.Error("Expected state update message")
				}
			}
			result := operation.RenderState()
			if result != test.expectedOutput {
				t.Errorf("Result was not expected.\nExpected:\n'%v'\n\nGot:\n'%v'", test.expectedOutput, result)
			}
			if operation.Done() != test.expectedDone {
				t.Errorf("Done was not as expected (expected %v)", test.expectedDone)
			}
		})
	}
}
