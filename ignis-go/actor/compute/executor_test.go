package compute

import (
	"errors"
	"testing"
	"time"

	"github.com/9triver/ignis/actor/functions"
	"github.com/9triver/ignis/object"
	"github.com/9triver/ignis/proto"
)

// mockFunction is a mock implementation of functions.Function for testing
type mockFunction struct {
	name      string
	params    []string
	callFunc  func(map[string]object.Interface) (object.Interface, error)
	timedFunc func(map[string]object.Interface) (time.Duration, object.Interface, error)
}

func (m *mockFunction) Name() string {
	return m.name
}

func (m *mockFunction) Params() []string {
	return m.params
}

func (m *mockFunction) Call(params map[string]object.Interface) (object.Interface, error) {
	if m.callFunc != nil {
		return m.callFunc(params)
	}
	return object.NewLocal("result", object.LangGo), nil
}

func (m *mockFunction) TimedCall(params map[string]object.Interface) (time.Duration, object.Interface, error) {
	if m.timedFunc != nil {
		return m.timedFunc(params)
	}
	// Default implementation
	start := time.Now()
	obj, err := m.Call(params)
	return time.Since(start), obj, err
}

func (m *mockFunction) Language() proto.Language {
	return proto.Language_LANG_GO
}

func TestNewExecutor(t *testing.T) {
	handler := &mockFunction{
		name:   "testFunc",
		params: []string{"param1", "param2"},
	}

	executor := NewExecutor(handler)
	
	if executor == nil {
		t.Fatal("NewExecutor returned nil")
	}
	
	if executor.handler != handler {
		t.Error("Executor handler not set correctly")
	}
	
	if executor.requests == nil {
		t.Error("Executor requests channel not initialized")
	}

	// Clean up
	executor.Close()
}

func TestExecutorDeps(t *testing.T) {
	expectedParams := []string{"param1", "param2", "param3"}
	handler := &mockFunction{
		name:   "testFunc",
		params: expectedParams,
	}

	executor := NewExecutor(handler)
	defer executor.Close()

	deps := executor.Deps()
	if len(deps) != len(expectedParams) {
		t.Errorf("Expected %d dependencies, got %d", len(expectedParams), len(deps))
	}

	for i, dep := range deps {
		if dep != expectedParams[i] {
			t.Errorf("Expected dependency %s at index %d, got %s", expectedParams[i], i, dep)
		}
	}
}

func TestExecutorCallSuccess(t *testing.T) {
	expectedResult := object.NewLocal("success", object.LangGo)
	handler := &mockFunction{
		name:   "testFunc",
		params: []string{"param1"},
		callFunc: func(params map[string]object.Interface) (object.Interface, error) {
			return expectedResult, nil
		},
	}

	executor := NewExecutor(handler)
	defer executor.Close()

	done := make(chan struct{})
	var resultObj object.Interface
	var resultErr error

	input := &ExecInput{
		SessionID: "test-session",
		Params: map[string]object.Interface{
			"param1": object.NewLocal("value1", object.LangGo),
		},
		Timed: false,
		OnDone: func(obj object.Interface, err error, duration time.Duration) {
			resultObj = obj
			resultErr = err
			close(done)
		},
	}

	executor.Requests() <- input

	select {
	case <-done:
		if resultErr != nil {
			t.Errorf("Expected no error, got %v", resultErr)
		}
		if resultObj != expectedResult {
			t.Error("Result object does not match expected")
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Test timed out waiting for execution")
	}
}

func TestExecutorCallError(t *testing.T) {
	expectedError := errors.New("execution failed")
	handler := &mockFunction{
		name:   "testFunc",
		params: []string{"param1"},
		callFunc: func(params map[string]object.Interface) (object.Interface, error) {
			return nil, expectedError
		},
	}

	executor := NewExecutor(handler)
	defer executor.Close()

	done := make(chan struct{})
	var resultErr error

	input := &ExecInput{
		SessionID: "test-session",
		Params: map[string]object.Interface{
			"param1": object.NewLocal("value1", object.LangGo),
		},
		Timed: false,
		OnDone: func(obj object.Interface, err error, duration time.Duration) {
			resultErr = err
			close(done)
		},
	}

	executor.Requests() <- input

	select {
	case <-done:
		if resultErr == nil {
			t.Error("Expected error, got nil")
		}
		if resultErr.Error() != expectedError.Error() {
			t.Errorf("Expected error %v, got %v", expectedError, resultErr)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Test timed out waiting for execution")
	}
}

func TestExecutorTimedCall(t *testing.T) {
	expectedDuration := 100 * time.Millisecond
	expectedResult := object.NewLocal("timed-result", object.LangGo)
	
	handler := &mockFunction{
		name:   "testFunc",
		params: []string{"param1"},
		timedFunc: func(params map[string]object.Interface) (time.Duration, object.Interface, error) {
			time.Sleep(expectedDuration)
			return expectedDuration, expectedResult, nil
		},
	}

	executor := NewExecutor(handler)
	defer executor.Close()

	done := make(chan struct{})
	var resultObj object.Interface
	var resultErr error
	var resultDuration time.Duration

	input := &ExecInput{
		SessionID: "test-session",
		Params: map[string]object.Interface{
			"param1": object.NewLocal("value1", object.LangGo),
		},
		Timed: true,
		OnDone: func(obj object.Interface, err error, duration time.Duration) {
			resultObj = obj
			resultErr = err
			resultDuration = duration
			close(done)
		},
	}

	executor.Requests() <- input

	select {
	case <-done:
		if resultErr != nil {
			t.Errorf("Expected no error, got %v", resultErr)
		}
		if resultObj != expectedResult {
			t.Error("Result object does not match expected")
		}
		if resultDuration < expectedDuration {
			t.Errorf("Expected duration >= %v, got %v", expectedDuration, resultDuration)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Test timed out waiting for execution")
	}
}

func TestExecutorMultipleRequests(t *testing.T) {
	callCount := 0
	handler := &mockFunction{
		name:   "testFunc",
		params: []string{"param1"},
		callFunc: func(params map[string]object.Interface) (object.Interface, error) {
			callCount++
			return object.NewLocal("result", object.LangGo), nil
		},
	}

	executor := NewExecutor(handler)
	defer executor.Close()

	numRequests := 5
	done := make(chan struct{}, numRequests)

	for i := 0; i < numRequests; i++ {
		input := &ExecInput{
			SessionID: "test-session",
			Params: map[string]object.Interface{
				"param1": object.NewLocal("value1", object.LangGo),
			},
			Timed: false,
			OnDone: func(obj object.Interface, err error, duration time.Duration) {
				done <- struct{}{}
			},
		}
		executor.Requests() <- input
	}

	// Wait for all requests to complete
	for i := 0; i < numRequests; i++ {
		select {
		case <-done:
			// Success
		case <-time.After(2 * time.Second):
			t.Fatalf("Timeout waiting for request %d", i+1)
		}
	}

	if callCount != numRequests {
		t.Errorf("Expected %d calls, got %d", numRequests, callCount)
	}
}

func TestExecutorClose(t *testing.T) {
	handler := &mockFunction{
		name:   "testFunc",
		params: []string{"param1"},
	}

	executor := NewExecutor(handler)
	executor.Close()

	// Attempting to send to closed channel should panic
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic when sending to closed channel")
		}
	}()

	executor.Requests() <- &ExecInput{}
}

func TestExecutorRequests(t *testing.T) {
	handler := &mockFunction{
		name:   "testFunc",
		params: []string{"param1"},
	}

	executor := NewExecutor(handler)
	defer executor.Close()

	requestsChan := executor.Requests()
	if requestsChan == nil {
		t.Fatal("Requests() returned nil channel")
	}

	// Verify it's a send-only channel by type
	var _ chan<- *ExecInput = requestsChan
}

func TestExecutorWithGoFunction(t *testing.T) {
	// Test with a real Go function
	type Input struct {
		X int
		Y int
	}

	type Output struct {
		Sum int
	}

	innerFunc := func(input Input) (Output, error) {
		return Output{Sum: input.X + input.Y}, nil
	}

	goFunc := functions.NewGo("add", innerFunc, object.LangGo)
	executor := NewExecutor(goFunc)
	defer executor.Close()

	done := make(chan struct{})
	var resultObj object.Interface
	var resultErr error

	input := &ExecInput{
		SessionID: "test-session",
		Params: map[string]object.Interface{
			"X": object.NewLocal(5, object.LangGo),
			"Y": object.NewLocal(3, object.LangGo),
		},
		Timed: false,
		OnDone: func(obj object.Interface, err error, duration time.Duration) {
			resultObj = obj
			resultErr = err
			close(done)
		},
	}

	executor.Requests() <- input

	select {
	case <-done:
		if resultErr != nil {
			t.Errorf("Expected no error, got %v", resultErr)
		}
		if resultObj == nil {
			t.Fatal("Result object is nil")
		}
		
		value, err := resultObj.Value()
		if err != nil {
			t.Fatalf("Failed to get value: %v", err)
		}
		
		output, ok := value.(Output)
		if !ok {
			t.Fatalf("Expected Output type, got %T", value)
		}
		
		if output.Sum != 8 {
			t.Errorf("Expected sum 8, got %d", output.Sum)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Test timed out waiting for execution")
	}
}

func TestExecutorWithNilContext(t *testing.T) {
	handler := &mockFunction{
		name:   "testFunc",
		params: []string{"param1"},
		callFunc: func(params map[string]object.Interface) (object.Interface, error) {
			return object.NewLocal("result", object.LangGo), nil
		},
	}

	executor := NewExecutor(handler)
	defer executor.Close()

	done := make(chan struct{})
	
	// Test with nil Context (which should be fine since executor doesn't use it directly)
	input := &ExecInput{
		Context:   nil,
		SessionID: "test-session",
		Params: map[string]object.Interface{
			"param1": object.NewLocal("value1", object.LangGo),
		},
		Timed: false,
		OnDone: func(obj object.Interface, err error, duration time.Duration) {
			close(done)
		},
	}

	executor.Requests() <- input

	select {
	case <-done:
		// Success
	case <-time.After(1 * time.Second):
		t.Fatal("Test timed out waiting for execution")
	}
}

