package tracker

import "testing"

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
		expectedFailed bool
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
			name: "Single job, succeeded",
			jobs: []struct {
				name    string
				state   jobState
				message string
				extra   []string
			}{
				{name: "job1", message: "testState", state: jobStateSuccess},
			},
			expectedOutput: "job1: [testState]",
			expectedDone:   true,
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
			expectedFailed: true,
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
			expectedFailed: true,
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
			result := operation.Render(0)
			if result != test.expectedOutput {
				t.Errorf("Result was not expected.\nExpected:\n'%v'\n\nGot:\n'%v'", test.expectedOutput, result)
			}
			if operation.Done() != test.expectedDone {
				t.Errorf("Done was not as expected (expected %v)", test.expectedDone)
			}
			if operation.Failed() != test.expectedFailed {
				t.Errorf("Failed was not as expected (expected %v)", test.expectedFailed)
			}
			operation.Close()
			select {
			case <-operation.StateUpdate():
			default:
				t.Error("Expected operation closed")
			}
		})
	}
}

func TestSubOperation(t *testing.T) {
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
		expectedFailed bool
	}{
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
			expectedOutput: "child:\n  job1: [testState]",
		},
		{
			name: "Single job, succeeded",
			jobs: []struct {
				name    string
				state   jobState
				message string
				extra   []string
			}{
				{name: "job1", message: "testState", state: jobStateSuccess},
			},
			expectedOutput: "child:\n  job1: [testState]",
			expectedDone:   true,
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
			expectedOutput: "child:\n  job1: [i]\n  job2: [s]\n  job3: [w]\n  job4: [f]",
			expectedDone:   true,
			expectedFailed: true,
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
			expectedOutput: "child:\n  job1: [i]\n  job2: [s]\n  job3: [w]",
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
			expectedOutput: "child:\n  job1: [i]\n  job4: [f]",
			expectedDone:   true,
			expectedFailed: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			operation := NewOperation()
			child := operation.GetOperation("child")
			for _, state := range test.jobs {
				job := child.GetJob(state.name)
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
			result := operation.Render(0)
			if result != test.expectedOutput {
				t.Errorf("Result was not expected.\nExpected:\n'%v'\n\nGot:\n'%v'", test.expectedOutput, result)
			}
			if operation.Done() != test.expectedDone {
				t.Errorf("Done was not as expected (expected %v)", test.expectedDone)
			}
			if operation.Failed() != test.expectedFailed {
				t.Errorf("Failed was not as expected (expected %v)", test.expectedFailed)
			}
			operation.Close()
			select {
			case <-operation.StateUpdate():
			default:
				t.Error("Expected operation closed")
			}
		})
	}
}
