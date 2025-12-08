package compute

import (
	"errors"
	"testing"
	"time"

	"github.com/9triver/ignis/actor/store"
	"github.com/9triver/ignis/object"
	"github.com/9triver/ignis/proto"
	"github.com/asynkron/protoactor-go/actor"
)

// mockStoreActor is a mock store actor for testing
type mockStoreActor struct {
	savedObjects    []object.Interface
	responses       []*proto.InvokeResponse
	receivedMsgChan chan interface{}
}

func (m *mockStoreActor) Receive(ctx actor.Context) {
	if m.receivedMsgChan != nil {
		m.receivedMsgChan <- ctx.Message()
	}

	switch msg := ctx.Message().(type) {
	case *store.SaveObject:
		m.savedObjects = append(m.savedObjects, msg.Value)
		if msg.Callback != nil {
			msg.Callback(ctx, &proto.Flow{
				ID:     msg.Value.GetID(),
				Source: &proto.StoreRef{ID: "mock-store"},
			})
		}
	case *proto.InvokeResponse:
		m.responses = append(m.responses, msg)
	}
}

func newMockStoreActor() *mockStoreActor {
	return &mockStoreActor{
		savedObjects:    make([]object.Interface, 0),
		responses:       make([]*proto.InvokeResponse, 0),
		receivedMsgChan: make(chan interface{}, 10),
	}
}

func TestNewSession(t *testing.T) {
	system := actor.NewActorSystem()
	defer system.Shutdown()

	mockStore := newMockStoreActor()
	storePID := system.Root.Spawn(actor.PropsFromProducer(func() actor.Actor {
		return mockStore
	}))

	handler := &mockFunction{
		name:   "testFunc",
		params: []string{"param1", "param2"},
	}
	executor := NewExecutor(handler)
	defer executor.Close()

	props := NewSession("test-session-1", storePID, executor)
	if props == nil {
		t.Fatal("NewSession returned nil props")
	}

	sessionPID := system.Root.Spawn(props)
	if sessionPID == nil {
		t.Fatal("Failed to spawn session actor")
	}

	system.Root.Stop(sessionPID)
}

func TestSessionDependencyTracking(t *testing.T) {
	system := actor.NewActorSystem()
	defer system.Shutdown()

	mockStore := newMockStoreActor()
	storePID := system.Root.Spawn(actor.PropsFromProducer(func() actor.Actor {
		return mockStore
	}))

	handler := &mockFunction{
		name:   "testFunc",
		params: []string{"param1", "param2"},
		callFunc: func(params map[string]object.Interface) (object.Interface, error) {
			return object.NewLocal("result", object.LangGo), nil
		},
	}
	executor := NewExecutor(handler)
	defer executor.Close()

	props := NewSession("test-session-2", storePID, executor)
	sessionPID := system.Root.Spawn(props)
	defer system.Root.Stop(sessionPID)

	// Send InvokeStart
	startMsg := &SessionStart{
		Info: &proto.ActorInfo{
			CalcLatency: 0,
			LinkLatency: 0,
		},
		ReplyTo: "test-controller",
	}
	system.Root.Send(sessionPID, startMsg)

	// Send first parameter
	invoke1 := &SessionInvoke{
		Param: "param1",
		Link:  10 * time.Millisecond,
		Value: object.NewLocal("value1", object.LangGo),
	}
	system.Root.Send(sessionPID, invoke1)

	// Send second parameter
	invoke2 := &SessionInvoke{
		Param: "param2",
		Link:  20 * time.Millisecond,
		Value: object.NewLocal("value2", object.LangGo),
	}
	system.Root.Send(sessionPID, invoke2)

	// Wait for execution and response
	time.Sleep(500 * time.Millisecond)

	// Verify response was sent
	if len(mockStore.responses) != 1 {
		t.Fatalf("Expected 1 response, got %d", len(mockStore.responses))
	}

	response := mockStore.responses[0]
	if response.SessionID != "test-session-2" {
		t.Errorf("Expected session ID 'test-session-2', got '%s'", response.SessionID)
	}
	if response.Target != "test-controller" {
		t.Errorf("Expected target 'test-controller', got '%s'", response.Target)
	}
	if response.Error != "" {
		t.Errorf("Expected no error, got '%s'", response.Error)
	}
	if response.Result == nil {
		t.Error("Expected result to be non-nil")
	}
}

func TestSessionStartBeforeParams(t *testing.T) {
	system := actor.NewActorSystem()
	defer system.Shutdown()

	mockStore := newMockStoreActor()
	storePID := system.Root.Spawn(actor.PropsFromProducer(func() actor.Actor {
		return mockStore
	}))

	handler := &mockFunction{
		name:   "testFunc",
		params: []string{"param1"},
		callFunc: func(params map[string]object.Interface) (object.Interface, error) {
			return object.NewLocal("result", object.LangGo), nil
		},
	}
	executor := NewExecutor(handler)
	defer executor.Close()

	props := NewSession("test-session-3", storePID, executor)
	sessionPID := system.Root.Spawn(props)
	defer system.Root.Stop(sessionPID)

	// Send start first
	startMsg := &SessionStart{
		Info:    nil,
		ReplyTo: "test-controller",
	}
	system.Root.Send(sessionPID, startMsg)

	// Then send parameter
	invoke := &SessionInvoke{
		Param: "param1",
		Link:  10 * time.Millisecond,
		Value: object.NewLocal("value1", object.LangGo),
	}
	system.Root.Send(sessionPID, invoke)

	// Wait for execution
	time.Sleep(500 * time.Millisecond)

	// Verify response was sent
	if len(mockStore.responses) != 1 {
		t.Fatalf("Expected 1 response, got %d", len(mockStore.responses))
	}
}

func TestSessionParamsBeforeStart(t *testing.T) {
	system := actor.NewActorSystem()
	defer system.Shutdown()

	mockStore := newMockStoreActor()
	storePID := system.Root.Spawn(actor.PropsFromProducer(func() actor.Actor {
		return mockStore
	}))

	handler := &mockFunction{
		name:   "testFunc",
		params: []string{"param1"},
		callFunc: func(params map[string]object.Interface) (object.Interface, error) {
			return object.NewLocal("result", object.LangGo), nil
		},
	}
	executor := NewExecutor(handler)
	defer executor.Close()

	props := NewSession("test-session-4", storePID, executor)
	sessionPID := system.Root.Spawn(props)
	defer system.Root.Stop(sessionPID)

	// Send parameter first
	invoke := &SessionInvoke{
		Param: "param1",
		Link:  10 * time.Millisecond,
		Value: object.NewLocal("value1", object.LangGo),
	}
	system.Root.Send(sessionPID, invoke)

	// Then send start
	startMsg := &SessionStart{
		Info:    nil,
		ReplyTo: "test-controller",
	}
	system.Root.Send(sessionPID, startMsg)

	// Wait for execution
	time.Sleep(500 * time.Millisecond)

	// Verify response was sent
	if len(mockStore.responses) != 1 {
		t.Fatalf("Expected 1 response, got %d", len(mockStore.responses))
	}
}

func TestSessionWithNoDependencies(t *testing.T) {
	system := actor.NewActorSystem()
	defer system.Shutdown()

	mockStore := newMockStoreActor()
	storePID := system.Root.Spawn(actor.PropsFromProducer(func() actor.Actor {
		return mockStore
	}))

	// Function with no parameters
	handler := &mockFunction{
		name:   "testFunc",
		params: []string{},
		callFunc: func(params map[string]object.Interface) (object.Interface, error) {
			return object.NewLocal("result", object.LangGo), nil
		},
	}
	executor := NewExecutor(handler)
	defer executor.Close()

	props := NewSession("test-session-5", storePID, executor)
	sessionPID := system.Root.Spawn(props)
	defer system.Root.Stop(sessionPID)

	// Send start (should trigger immediate execution)
	startMsg := &SessionStart{
		Info:    nil,
		ReplyTo: "test-controller",
	}
	system.Root.Send(sessionPID, startMsg)

	// Wait for execution
	time.Sleep(500 * time.Millisecond)

	// Verify response was sent
	if len(mockStore.responses) != 1 {
		t.Fatalf("Expected 1 response, got %d", len(mockStore.responses))
	}
}

func TestSessionExecutionError(t *testing.T) {
	system := actor.NewActorSystem()
	defer system.Shutdown()

	mockStore := newMockStoreActor()
	storePID := system.Root.Spawn(actor.PropsFromProducer(func() actor.Actor {
		return mockStore
	}))

	handler := &mockFunction{
		name:   "testFunc",
		params: []string{"param1"},
		callFunc: func(params map[string]object.Interface) (object.Interface, error) {
			return nil, errors.New("execution error")
		},
	}
	executor := NewExecutor(handler)
	defer executor.Close()

	props := NewSession("test-session-6", storePID, executor)
	sessionPID := system.Root.Spawn(props)
	defer system.Root.Stop(sessionPID)

	// Send parameter and start
	invoke := &SessionInvoke{
		Param: "param1",
		Link:  10 * time.Millisecond,
		Value: object.NewLocal("value1", object.LangGo),
	}
	system.Root.Send(sessionPID, invoke)

	startMsg := &SessionStart{
		Info:    nil,
		ReplyTo: "test-controller",
	}
	system.Root.Send(sessionPID, startMsg)

	// Wait for execution
	time.Sleep(500 * time.Millisecond)

	// Verify error response was sent
	if len(mockStore.responses) != 1 {
		t.Fatalf("Expected 1 response, got %d", len(mockStore.responses))
	}

	response := mockStore.responses[0]
	if response.Error == "" {
		t.Error("Expected error in response, got empty string")
	}
	if response.Result != nil {
		t.Error("Expected nil result on error")
	}
}

func TestSessionWithActorInfo(t *testing.T) {
	system := actor.NewActorSystem()
	defer system.Shutdown()

	mockStore := newMockStoreActor()
	storePID := system.Root.Spawn(actor.PropsFromProducer(func() actor.Actor {
		return mockStore
	}))

	handler := &mockFunction{
		name:   "testFunc",
		params: []string{"param1"},
		callFunc: func(params map[string]object.Interface) (object.Interface, error) {
			time.Sleep(50 * time.Millisecond) // Simulate some work
			return object.NewLocal("result", object.LangGo), nil
		},
	}
	executor := NewExecutor(handler)
	defer executor.Close()

	props := NewSession("test-session-7", storePID, executor)
	sessionPID := system.Root.Spawn(props)
	defer system.Root.Stop(sessionPID)

	// Send with ActorInfo for latency measurement
	startMsg := &SessionStart{
		Info: &proto.ActorInfo{
			CalcLatency: 100,
			LinkLatency: 50,
		},
		ReplyTo: "test-controller",
	}
	system.Root.Send(sessionPID, startMsg)

	invoke := &SessionInvoke{
		Param: "param1",
		Link:  20 * time.Millisecond,
		Value: object.NewLocal("value1", object.LangGo),
	}
	system.Root.Send(sessionPID, invoke)

	// Wait for execution
	time.Sleep(500 * time.Millisecond)

	// Verify response has updated ActorInfo
	if len(mockStore.responses) != 1 {
		t.Fatalf("Expected 1 response, got %d", len(mockStore.responses))
	}

	response := mockStore.responses[0]
	if response.Info == nil {
		t.Fatal("Expected ActorInfo in response")
	}

	// CalcLatency should be updated (averaged with execution time)
	if response.Info.CalcLatency == 0 {
		t.Error("Expected non-zero CalcLatency")
	}

	// LinkLatency should be updated (averaged with link time)
	if response.Info.LinkLatency == 0 {
		t.Error("Expected non-zero LinkLatency")
	}
}

func TestSessionMultipleParameters(t *testing.T) {
	system := actor.NewActorSystem()
	defer system.Shutdown()

	mockStore := newMockStoreActor()
	storePID := system.Root.Spawn(actor.PropsFromProducer(func() actor.Actor {
		return mockStore
	}))

	receivedParams := make(map[string]object.Interface)
	handler := &mockFunction{
		name:   "testFunc",
		params: []string{"x", "y", "z"},
		callFunc: func(params map[string]object.Interface) (object.Interface, error) {
			for k, v := range params {
				receivedParams[k] = v
			}
			return object.NewLocal("result", object.LangGo), nil
		},
	}
	executor := NewExecutor(handler)
	defer executor.Close()

	props := NewSession("test-session-8", storePID, executor)
	sessionPID := system.Root.Spawn(props)
	defer system.Root.Stop(sessionPID)

	// Send multiple parameters
	params := map[string]string{
		"x": "value-x",
		"y": "value-y",
		"z": "value-z",
	}

	for param, value := range params {
		invoke := &SessionInvoke{
			Param: param,
			Link:  10 * time.Millisecond,
			Value: object.NewLocal(value, object.LangGo),
		}
		system.Root.Send(sessionPID, invoke)
	}

	// Send start
	startMsg := &SessionStart{
		Info:    nil,
		ReplyTo: "test-controller",
	}
	system.Root.Send(sessionPID, startMsg)

	// Wait for execution
	time.Sleep(500 * time.Millisecond)

	// Verify all parameters were received
	if len(receivedParams) != 3 {
		t.Fatalf("Expected 3 parameters, got %d", len(receivedParams))
	}

	for param, expectedValue := range params {
		if obj, ok := receivedParams[param]; !ok {
			t.Errorf("Missing parameter %s", param)
		} else {
			value, err := obj.Value()
			if err != nil {
				t.Errorf("Failed to get value for param %s: %v", param, err)
			}
			if value != expectedValue {
				t.Errorf("Expected value %s for param %s, got %v", expectedValue, param, value)
			}
		}
	}
}

func TestSessionStopMessage(t *testing.T) {
	system := actor.NewActorSystem()
	defer system.Shutdown()

	mockStore := newMockStoreActor()
	storePID := system.Root.Spawn(actor.PropsFromProducer(func() actor.Actor {
		return mockStore
	}))

	handler := &mockFunction{
		name:   "testFunc",
		params: []string{"param1"},
	}
	executor := NewExecutor(handler)

	props := NewSession("test-session-9", storePID, executor)
	sessionPID := system.Root.Spawn(props)

	// Stop the session
	system.Root.Stop(sessionPID)

	// Wait a bit for stop to process
	time.Sleep(100 * time.Millisecond)

	// Note: We can't easily verify executor.Close() was called from here,
	// but we test that the actor stops without error
}

func TestSessionLinkAccumulation(t *testing.T) {
	system := actor.NewActorSystem()
	defer system.Shutdown()

	mockStore := newMockStoreActor()
	storePID := system.Root.Spawn(actor.PropsFromProducer(func() actor.Actor {
		return mockStore
	}))

	handler := &mockFunction{
		name:   "testFunc",
		params: []string{"param1", "param2"},
		callFunc: func(params map[string]object.Interface) (object.Interface, error) {
			return object.NewLocal("result", object.LangGo), nil
		},
	}
	executor := NewExecutor(handler)
	defer executor.Close()

	props := NewSession("test-session-10", storePID, executor)
	sessionPID := system.Root.Spawn(props)
	defer system.Root.Stop(sessionPID)

	link1 := 10 * time.Millisecond
	link2 := 20 * time.Millisecond

	// Send parameters with different link times
	invoke1 := &SessionInvoke{
		Param: "param1",
		Link:  link1,
		Value: object.NewLocal("value1", object.LangGo),
	}
	system.Root.Send(sessionPID, invoke1)

	invoke2 := &SessionInvoke{
		Param: "param2",
		Link:  link2,
		Value: object.NewLocal("value2", object.LangGo),
	}
	system.Root.Send(sessionPID, invoke2)

	// Send start with ActorInfo
	startMsg := &SessionStart{
		Info: &proto.ActorInfo{
			CalcLatency: 0,
			LinkLatency: 0,
		},
		ReplyTo: "test-controller",
	}
	system.Root.Send(sessionPID, startMsg)

	// Wait for execution
	time.Sleep(500 * time.Millisecond)

	// Verify link latency was accumulated
	if len(mockStore.responses) != 1 {
		t.Fatalf("Expected 1 response, got %d", len(mockStore.responses))
	}

	response := mockStore.responses[0]
	if response.Info == nil {
		t.Fatal("Expected ActorInfo in response")
	}

	// LinkLatency should be the average of 0 (initial) and accumulated link time
	expectedLink := int64(link1 + link2)
	if response.Info.LinkLatency != expectedLink/2 {
		t.Errorf("Expected LinkLatency %d, got %d", expectedLink/2, response.Info.LinkLatency)
	}
}

