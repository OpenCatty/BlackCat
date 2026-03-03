package dashboard

import (
	"fmt"
	"net/http"
	"sync"
)

// QRBroadcaster manages SSE subscriptions for WhatsApp QR codes.
// It is a lightweight separate broadcaster so QR events don't pollute the main event stream.
type QRBroadcaster struct {
	mu          sync.RWMutex
	subscribers map[chan string]struct{}
}

func NewQRBroadcaster() *QRBroadcaster {
	return &QRBroadcaster{
		subscribers: make(map[chan string]struct{}),
	}
}

func (q *QRBroadcaster) Send(qrCode string) {
	q.mu.RLock()
	subs := make([]chan string, 0, len(q.subscribers))
	for ch := range q.subscribers {
		subs = append(subs, ch)
	}
	q.mu.RUnlock()

	for _, ch := range subs {
		select {
		case ch <- qrCode:
		default:
		}
	}
}

func (q *QRBroadcaster) Subscribe() (<-chan string, func()) {
	ch := make(chan string, 8)

	q.mu.Lock()
	q.subscribers[ch] = struct{}{}
	q.mu.Unlock()

	unsubscribe := func() {
		q.mu.Lock()
		delete(q.subscribers, ch)
		q.mu.Unlock()
		close(ch)
	}

	return ch, unsubscribe
}

func (s *Server) handleQRStream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ch, unsub := s.qrBroadcaster.Subscribe()
	defer unsub()

	fmt.Fprintf(w, ": keepalive\n\n")
	flusher.Flush()

	for {
		select {
		case <-r.Context().Done():
			return
		case qrCode, ok := <-ch:
			if !ok {
				return
			}
			fmt.Fprintf(w, "event: qr\ndata: %s\n\n", qrCode)
			flusher.Flush()
		}
	}
}
