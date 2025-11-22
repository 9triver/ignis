package compute

import (
	"testing"
	"time"

	"github.com/9triver/ignis/object"
	"github.com/9triver/ignis/proto"
	"github.com/asynkron/protoactor-go/actor"
)

func TestNewActor(t *testing.T) {
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

	props := NewActor("test-actor", handler, storePID)
	if props == nil {
		t.Fatal("NewActor returned nil props")
	}

	actorPID := system.Root.Spawn(props)
	if actorPID == nil {
		t.Fatal("Failed to spawn compute actor")
	}

	system.Root.Stop(actorPID)
}

func TestActorOnInvokeStart(t *testing.T) {
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

	props := NewActor("test-actor", handler, storePID)
	actorPID := system.Root.Spawn(props)
	defer system.Root.Stop(actorPID)

	// Send InvokeStart
	invokeStart := &proto.InvokeStart{
		Info: &proto.ActorInfo{
			CalcLatency: 0,
			LinkLatency: 0,
		},
		SessionID: "session-1",
		ReplyTo:   "test-controller",
	}

	system.Root.Send(actorPID, invokeStart)

	// Wait for session creation
	time.Sleep(200 * time.Millisecond)

	// Session should have been created
	// We can't directly access the sessions map, but we can verify by sending an invoke
	invoke := &proto.Invoke{
		Target:    "test-actor",
		SessionID: "session-1",
		Param:     "param1",
		Value: &proto.Flow{
			ID:     "test-obj-1",
			Source: &proto.StoreRef{ID: "mock-store", PID: storePID},
		},
	}

	// Add object to mock store
	mockStore.savedObjects = append(mockStore.savedObjects, object.NewLocal("value1", object.LangGo))

	system.Root.Send(actorPID, invoke)

	// Wait for processing
	time.Sleep(500 * time.Millisecond)

	// Should receive response
	if len(mockStore.responses) < 1 {
		t.Fatalf("Expected at least 1 response, got %d", len(mockStore.responses))
	}
}

func TestActorOnInvoke(t *testing.T) {
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

	props := NewActor("test-actor", handler, storePID)
	actorPID := system.Root.Spawn(props)
	defer system.Root.Stop(actorPID)

	// Create and save an object in the mock store
	testObj := object.NewLocal("test-value", object.LangGo)
	mockStore.savedObjects = append(mockStore.savedObjects, testObj)

	// Send InvokeStart first
	invokeStart := &proto.InvokeStart{
		SessionID: "session-2",
		ReplyTo:   "test-controller",
	}
	system.Root.Send(actorPID, invokeStart)

	time.Sleep(100 * time.Millisecond)

	// Send Invoke
	invoke := &proto.Invoke{
		Target:    "test-actor",
		SessionID: "session-2",
		Param:     "param1",
		Value: &proto.Flow{
			ID:     testObj.GetID(),
			Source: &proto.StoreRef{ID: "mock-store", PID: storePID},
		},
	}

	system.Root.Send(actorPID, invoke)

	// Wait for processing
	time.Sleep(500 * time.Millisecond)

	// Verify response was sent
	if len(mockStore.responses) < 1 {
		t.Fatalf("Expected at least 1 response, got %d", len(mockStore.responses))
	}
}

func TestActorMultipleSessions(t *testing.T) {
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

	props := NewActor("test-actor", handler, storePID)
	actorPID := system.Root.Spawn(props)
	defer system.Root.Stop(actorPID)

	// Create multiple sessions
	sessionIDs := []string{"session-3a", "session-3b", "session-3c"}

	for _, sessionID := range sessionIDs {
		// Send InvokeStart
		invokeStart := &proto.InvokeStart{
			SessionID: sessionID,
			ReplyTo:   "test-controller",
		}
		system.Root.Send(actorPID, invokeStart)

		time.Sleep(50 * time.Millisecond)

		// Send Invoke
		testObj := object.NewLocal("test-value-"+sessionID, object.LangGo)
		mockStore.savedObjects = append(mockStore.savedObjects, testObj)

		invoke := &proto.Invoke{
			Target:    "test-actor",
			SessionID: sessionID,
			Param:     "param1",
			Value: &proto.Flow{
				ID:     testObj.GetID(),
				Source: &proto.StoreRef{ID: "mock-store", PID: storePID},
			},
		}
		system.Root.Send(actorPID, invoke)
	}

	// Wait for all processing
	time.Sleep(1 * time.Second)

	// Verify responses for all sessions
	if len(mockStore.responses) < len(sessionIDs) {
		t.Fatalf("Expected at least %d responses, got %d", len(sessionIDs), len(mockStore.responses))
	}
}

func TestActorInvokeBeforeStart(t *testing.T) {
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

	props := NewActor("test-actor", handler, storePID)
	actorPID := system.Root.Spawn(props)
	defer system.Root.Stop(actorPID)

	// Send Invoke before InvokeStart (should still create session)
	testObj := object.NewLocal("test-value", object.LangGo)
	mockStore.savedObjects = append(mockStore.savedObjects, testObj)

	invoke := &proto.Invoke{
		Target:    "test-actor",
		SessionID: "session-4",
		Param:     "param1",
		Value: &proto.Flow{
			ID:     testObj.GetID(),
			Source: &proto.StoreRef{ID: "mock-store", PID: storePID},
		},
	}
	system.Root.Send(actorPID, invoke)

	time.Sleep(200 * time.Millisecond)

	// Now send InvokeStart
	invokeStart := &proto.InvokeStart{
		SessionID: "session-4",
		ReplyTo:   "test-controller",
	}
	system.Root.Send(actorPID, invokeStart)

	// Wait for processing
	time.Sleep(500 * time.Millisecond)

	// Verify response was sent
	if len(mockStore.responses) < 1 {
		t.Fatalf("Expected at least 1 response, got %d", len(mockStore.responses))
	}
}

func TestActorNewSessionCreation(t *testing.T) {
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

	props := NewActor("test-actor", handler, storePID)
	actorPID := system.Root.Spawn(props)
	defer system.Root.Stop(actorPID)

	// Send InvokeStart - should create new session
	invokeStart := &proto.InvokeStart{
		SessionID: "new-session",
		ReplyTo:   "test-controller",
	}
	system.Root.Send(actorPID, invokeStart)

	time.Sleep(200 * time.Millisecond)

	// Session should exist and handle subsequent messages
	// We verify this by sending invoke to same session
	testObj := object.NewLocal("test-value", object.LangGo)
	mockStore.savedObjects = append(mockStore.savedObjects, testObj)

	invoke := &proto.Invoke{
		Target:    "test-actor",
		SessionID: "new-session",
		Param:     "param1",
		Value: &proto.Flow{
			ID:     testObj.GetID(),
			Source: &proto.StoreRef{ID: "mock-store", PID: storePID},
		},
	}
	system.Root.Send(actorPID, invoke)

	time.Sleep(500 * time.Millisecond)

	// Should successfully process
	if len(mockStore.responses) < 1 {
		t.Fatalf("Expected at least 1 response, got %d", len(mockStore.responses))
	}
}

func TestActorWithMultipleParams(t *testing.T) {
	system := actor.NewActorSystem()
	defer system.Shutdown()

	mockStore := newMockStoreActor()
	storePID := system.Root.Spawn(actor.PropsFromProducer(func() actor.Actor {
		return mockStore
	}))

	handler := &mockFunction{
		name:   "testFunc",
		params: []string{"x", "y"},
		callFunc: func(params map[string]object.Interface) (object.Interface, error) {
			return object.NewLocal("result", object.LangGo), nil
		},
	}

	props := NewActor("test-actor", handler, storePID)
	actorPID := system.Root.Spawn(props)
	defer system.Root.Stop(actorPID)

	// Send InvokeStart
	invokeStart := &proto.InvokeStart{
		SessionID: "session-multi",
		ReplyTo:   "test-controller",
	}
	system.Root.Send(actorPID, invokeStart)

	time.Sleep(100 * time.Millisecond)

	// Send multiple parameters
	for _, param := range []string{"x", "y"} {
		testObj := object.NewLocal("value-"+param, object.LangGo)
		mockStore.savedObjects = append(mockStore.savedObjects, testObj)

		invoke := &proto.Invoke{
			Target:    "test-actor",
			SessionID: "session-multi",
			Param:     param,
			Value: &proto.Flow{
				ID:     testObj.GetID(),
				Source: &proto.StoreRef{ID: "mock-store", PID: storePID},
			},
		}
		system.Root.Send(actorPID, invoke)
		time.Sleep(50 * time.Millisecond)
	}

	// Wait for processing
	time.Sleep(500 * time.Millisecond)

	// Verify response was sent
	if len(mockStore.responses) < 1 {
		t.Fatalf("Expected at least 1 response, got %d", len(mockStore.responses))
	}

	response := mockStore.responses[len(mockStore.responses)-1]
	if response.SessionID != "session-multi" {
		t.Errorf("Expected session ID 'session-multi', got '%s'", response.SessionID)
	}
}

func TestActorReuseSessionID(t *testing.T) {
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

	props := NewActor("test-actor", handler, storePID)
	actorPID := system.Root.Spawn(props)
	defer system.Root.Stop(actorPID)

	sessionID := "reuse-session"

	// First InvokeStart
	invokeStart := &proto.InvokeStart{
		SessionID: sessionID,
		ReplyTo:   "test-controller",
	}
	system.Root.Send(actorPID, invokeStart)

	time.Sleep(100 * time.Millisecond)

	// Second InvokeStart with same session ID (should reuse existing session)
	invokeStart2 := &proto.InvokeStart{
		SessionID: sessionID,
		ReplyTo:   "test-controller-2",
	}
	system.Root.Send(actorPID, invokeStart2)

	time.Sleep(100 * time.Millisecond)

	// Send Invoke
	testObj := object.NewLocal("test-value", object.LangGo)
	mockStore.savedObjects = append(mockStore.savedObjects, testObj)

	invoke := &proto.Invoke{
		Target:    "test-actor",
		SessionID: sessionID,
		Param:     "param1",
		Value: &proto.Flow{
			ID:     testObj.GetID(),
			Source: &proto.StoreRef{ID: "mock-store", PID: storePID},
		},
	}
	system.Root.Send(actorPID, invoke)

	// Wait for processing
	time.Sleep(500 * time.Millisecond)

	// Should have responses (may have multiple if both starts triggered)
	if len(mockStore.responses) < 1 {
		t.Fatalf("Expected at least 1 response, got %d", len(mockStore.responses))
	}
}

func TestActorWithActorInfo(t *testing.T) {
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
			time.Sleep(50 * time.Millisecond)
			return object.NewLocal("result", object.LangGo), nil
		},
	}

	props := NewActor("test-actor", handler, storePID)
	actorPID := system.Root.Spawn(props)
	defer system.Root.Stop(actorPID)

	// Send InvokeStart with ActorInfo
	invokeStart := &proto.InvokeStart{
		Info: &proto.ActorInfo{
			CalcLatency: 100,
			LinkLatency: 50,
		},
		SessionID: "info-session",
		ReplyTo:   "test-controller",
	}
	system.Root.Send(actorPID, invokeStart)

	time.Sleep(100 * time.Millisecond)

	// Send Invoke
	testObj := object.NewLocal("test-value", object.LangGo)
	mockStore.savedObjects = append(mockStore.savedObjects, testObj)

	invoke := &proto.Invoke{
		Target:    "test-actor",
		SessionID: "info-session",
		Param:     "param1",
		Value: &proto.Flow{
			ID:     testObj.GetID(),
			Source: &proto.StoreRef{ID: "mock-store", PID: storePID},
		},
	}
	system.Root.Send(actorPID, invoke)

	// Wait for processing
	time.Sleep(500 * time.Millisecond)

	// Verify response has ActorInfo
	if len(mockStore.responses) < 1 {
		t.Fatalf("Expected at least 1 response, got %d", len(mockStore.responses))
	}

	response := mockStore.responses[len(mockStore.responses)-1]
	if response.Info == nil {
		t.Error("Expected ActorInfo in response")
	}
}

func TestActorIgnoresUnknownMessages(t *testing.T) {
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

	props := NewActor("test-actor", handler, storePID)
	actorPID := system.Root.Spawn(props)
	defer system.Root.Stop(actorPID)

	// Send unknown message type
	type UnknownMessage struct {
		Data string
	}
	system.Root.Send(actorPID, &UnknownMessage{Data: "test"})

	// Wait a bit
	time.Sleep(200 * time.Millisecond)

	// Actor should still be alive and handle valid messages
	invokeStart := &proto.InvokeStart{
		SessionID: "after-unknown",
		ReplyTo:   "test-controller",
	}
	system.Root.Send(actorPID, invokeStart)

	time.Sleep(100 * time.Millisecond)

	// Should handle this fine
	testObj := object.NewLocal("test-value", object.LangGo)
	mockStore.savedObjects = append(mockStore.savedObjects, testObj)

	invoke := &proto.Invoke{
		Target:    "test-actor",
		SessionID: "after-unknown",
		Param:     "param1",
		Value: &proto.Flow{
			ID:     testObj.GetID(),
			Source: &proto.StoreRef{ID: "mock-store", PID: storePID},
		},
	}
	system.Root.Send(actorPID, invoke)

	time.Sleep(500 * time.Millisecond)

	// Should still process normally
	if len(mockStore.responses) < 1 {
		t.Fatalf("Expected at least 1 response, got %d", len(mockStore.responses))
	}
}

func TestActorConcurrentSessions(t *testing.T) {
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
			time.Sleep(100 * time.Millisecond) // Simulate work
			return object.NewLocal("result", object.LangGo), nil
		},
	}

	props := NewActor("test-actor", handler, storePID)
	actorPID := system.Root.Spawn(props)
	defer system.Root.Stop(actorPID)

	numSessions := 5

	// Start all sessions concurrently
	for i := 0; i < numSessions; i++ {
		sessionID := time.Now().Format("session-concurrent-") + string(rune(i+'a'))

		// Send InvokeStart
		invokeStart := &proto.InvokeStart{
			SessionID: sessionID,
			ReplyTo:   "test-controller",
		}
		system.Root.Send(actorPID, invokeStart)

		// Send Invoke
		testObj := object.NewLocal("test-value-"+sessionID, object.LangGo)
		mockStore.savedObjects = append(mockStore.savedObjects, testObj)

		invoke := &proto.Invoke{
			Target:    "test-actor",
			SessionID: sessionID,
			Param:     "param1",
			Value: &proto.Flow{
				ID:     testObj.GetID(),
				Source: &proto.StoreRef{ID: "mock-store", PID: storePID},
			},
		}
		system.Root.Send(actorPID, invoke)
	}

	// Wait for all to complete
	time.Sleep(2 * time.Second)

	// Should have responses for all sessions
	if len(mockStore.responses) < numSessions {
		t.Fatalf("Expected at least %d responses, got %d", numSessions, len(mockStore.responses))
	}
}

