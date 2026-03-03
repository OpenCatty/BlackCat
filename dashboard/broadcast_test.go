package dashboard

import (
	"testing"
	"time"
)

// TestBroadcaster_SubscribeReceive tests basic subscription and event reception.
func TestBroadcaster_SubscribeReceive(t *testing.T) {
	b := NewBroadcaster()
	ch, unsub := b.Subscribe()
	defer unsub()

	b.Send("test event")

	select {
	case msg := <-ch:
		if msg != "test event" {
			t.Errorf("expected 'test event', got %q", msg)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for event")
	}
}

// TestBroadcaster_Unsubscribe tests that unsubscribing stops event reception.
func TestBroadcaster_Unsubscribe(t *testing.T) {
	b := NewBroadcaster()
	ch, unsub := b.Subscribe()

	unsub()

	// Channel should be closed after unsubscribe
	select {
	case _, ok := <-ch:
		if ok {
			t.Fatal("expected channel to be closed after unsubscribe")
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for channel close")
	}

	// Sending after unsubscribe should not panic
	b.Send("event after unsubscribe")
}

// TestBroadcaster_MultipleSubscribers tests that multiple subscribers all receive events.
func TestBroadcaster_MultipleSubscribers(t *testing.T) {
	b := NewBroadcaster()

	ch1, unsub1 := b.Subscribe()
	ch2, unsub2 := b.Subscribe()
	ch3, unsub3 := b.Subscribe()
	defer unsub1()
	defer unsub2()
	defer unsub3()

	b.Send("broadcast message")

	// All three should receive the message
	timeout := time.Second
	for i, ch := range []<-chan string{ch1, ch2, ch3} {
		select {
		case msg := <-ch:
			if msg != "broadcast message" {
				t.Errorf("subscriber %d got %q, expected 'broadcast message'", i+1, msg)
			}
		case <-time.After(timeout):
			t.Fatalf("subscriber %d timeout", i+1)
		}
	}
}

// TestBroadcaster_NonBlocking tests that Send does not block on slow subscribers.
func TestBroadcaster_NonBlocking(t *testing.T) {
	b := NewBroadcaster()
	ch, unsub := b.Subscribe()
	defer unsub()

	// Fill the buffer (capacity 8)
	for i := 0; i < 8; i++ {
		b.Send("event " + string(rune(i)))
	}

	// Send again - should not block even though buffer is full
	done := make(chan bool)
	go func() {
		b.Send("extra event")
		done <- true
	}()

	select {
	case <-done:
		// Success: Send returned without blocking
	case <-time.After(time.Second):
		t.Fatal("Send blocked on full buffer")
	}

	// Verify we can still read from the buffer
	select {
	case msg := <-ch:
		if msg == "" {
			t.Error("expected to receive an event")
		}
	case <-time.After(time.Second):
		t.Fatal("timeout reading from channel")
	}
}

// TestBroadcaster_ZeroSubscribers tests that Send doesn't panic with no subscribers.
func TestBroadcaster_ZeroSubscribers(t *testing.T) {
	b := NewBroadcaster()

	// Should not panic
	b.Send("event with no subscribers")
	b.Send("another event")
}
