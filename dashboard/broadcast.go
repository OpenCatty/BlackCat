package dashboard

import "sync"

// Broadcaster manages SSE subscriptions and broadcasts events to multiple clients.
type Broadcaster struct {
	mu          sync.RWMutex
	subscribers map[chan string]struct{}
}

// NewBroadcaster creates a new Broadcaster instance.
func NewBroadcaster() *Broadcaster {
	return &Broadcaster{
		subscribers: make(map[chan string]struct{}),
	}
}

// Subscribe returns a channel to receive events and an unsubscribe function.
// The returned channel has a buffer capacity of 8 to prevent blocking.
func (b *Broadcaster) Subscribe() (<-chan string, func()) {
	ch := make(chan string, 8)

	b.mu.Lock()
	b.subscribers[ch] = struct{}{}
	b.mu.Unlock()

	unsubscribe := func() {
		b.mu.Lock()
		delete(b.subscribers, ch)
		b.mu.Unlock()
		close(ch)
	}

	return ch, unsubscribe
}

// Send broadcasts an event to all subscribers using non-blocking sends.
// Slow subscribers will not block the send; their buffered channels will drop events if full.
func (b *Broadcaster) Send(event string) {
	b.mu.RLock()
	subs := make([]chan string, 0, len(b.subscribers))
	for ch := range b.subscribers {
		subs = append(subs, ch)
	}
	b.mu.RUnlock()

	for _, ch := range subs {
		select {
		case ch <- event:
		default:
			// Non-blocking: skip this subscriber if buffer is full
		}
	}
}
