package hooks

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

func TestHookRegistryPreHookStopsOnFirstError(t *testing.T) {
	registry := NewHookRegistry()
	errFirst := errors.New("first error")

	called := 0
	registry.Register(PreChat, func(ctx *HookContext) error {
		called++
		return errFirst
	})
	registry.Register(PreChat, func(ctx *HookContext) error {
		called++
		return nil
	})

	err := registry.Fire(context.Background(), PreChat, &HookContext{})
	if !errors.Is(err, errFirst) {
		t.Fatalf("expected first error, got %v", err)
	}
	if called != 1 {
		t.Fatalf("expected one handler call, got %d", called)
	}
}

func TestHookRegistryPostHookRunsAllHandlers(t *testing.T) {
	registry := NewHookRegistry()
	errOne := errors.New("post hook error")

	called := 0
	registry.Register(PostChat, func(ctx *HookContext) error {
		called++
		return errOne
	})
	registry.Register(PostChat, func(ctx *HookContext) error {
		called++
		return nil
	})

	err := registry.Fire(context.Background(), PostChat, &HookContext{})
	if !errors.Is(err, errOne) {
		t.Fatalf("expected collected post-hook error, got %v", err)
	}
	if called != 2 {
		t.Fatalf("expected all handlers to run, got %d", called)
	}
}

func TestHookRegistryRecoversPanics(t *testing.T) {
	registry := NewHookRegistry()

	called := 0
	registry.Register(PostToolExec, func(ctx *HookContext) error {
		panic("boom")
	})
	registry.Register(PostToolExec, func(ctx *HookContext) error {
		called++
		return nil
	})

	err := registry.Fire(context.Background(), PostToolExec, &HookContext{})
	if err == nil {
		t.Fatal("expected panic recovery error, got nil")
	}
	if called != 1 {
		t.Fatalf("expected non-panicking handler to still run, got %d", called)
	}
}

func TestHookRegistryPriorityOrdering(t *testing.T) {
	registry := NewHookRegistry()
	var order []int

	registry.RegisterWithPriority(PostChat, 200, func(ctx *HookContext) error {
		order = append(order, 200)
		return nil
	})
	registry.RegisterWithPriority(PostChat, 10, func(ctx *HookContext) error {
		order = append(order, 10)
		return nil
	})
	registry.RegisterWithPriority(PostChat, 100, func(ctx *HookContext) error {
		order = append(order, 100)
		return nil
	})

	if err := registry.Fire(context.Background(), PostChat, &HookContext{}); err != nil {
		t.Fatalf("expected no error firing prioritized handlers, got %v", err)
	}

	expected := []int{10, 100, 200}
	if !reflect.DeepEqual(order, expected) {
		t.Fatalf("unexpected handler order: got %v, want %v", order, expected)
	}
}

func TestHookRegistryMessageLifecycleEvents(t *testing.T) {
	registry := NewHookRegistry()
	events := []HookEvent{MessageReceived, MessageSending, MessageSent}

	for _, event := range events {
		event := event
		called := false
		hctx := &HookContext{}

		registry.Register(event, func(ctx *HookContext) error {
			called = true
			if ctx.Event != event {
				t.Fatalf("expected event %s, got %s", event, ctx.Event)
			}
			return nil
		})

		if err := registry.Fire(context.Background(), event, hctx); err != nil {
			t.Fatalf("expected no error for %s, got %v", event, err)
		}
		if !called {
			t.Fatalf("expected handler to be called for %s", event)
		}
	}
}

func TestHookRegistryRegisterUsesDefaultPriority(t *testing.T) {
	registry := NewHookRegistry()
	var order []int

	registry.Register(PostChat, func(ctx *HookContext) error {
		order = append(order, 100)
		return nil
	})
	registry.RegisterWithPriority(PostChat, 50, func(ctx *HookContext) error {
		order = append(order, 50)
		return nil
	})
	registry.RegisterWithPriority(PostChat, 150, func(ctx *HookContext) error {
		order = append(order, 150)
		return nil
	})

	if err := registry.Fire(context.Background(), PostChat, &HookContext{}); err != nil {
		t.Fatalf("expected no error firing default priority handlers, got %v", err)
	}

	expected := []int{50, 100, 150}
	if !reflect.DeepEqual(order, expected) {
		t.Fatalf("unexpected handler order with default priority: got %v, want %v", order, expected)
	}
}
