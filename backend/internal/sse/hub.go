// Package sse is the realtime seat-map channel (F11): a hub fanning seat
// changes out per showtime, plus short-lived connection tokens.
package sse

import "sync"

// SeatUpdate is one seat status change. ID is the showtime_seat id.
type SeatUpdate struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

// SeatEvent is a batch of seat changes of one showtime.
type SeatEvent struct {
	ShowtimeID string       `json:"showtime_id"`
	Seats      []SeatUpdate `json:"seats"`
}

// clientBuffer bounds queued events per connection; a client that falls this
// far behind is dropped instead of slowing everyone down (R-S4).
const clientBuffer = 64

type client struct {
	ch   chan SeatEvent
	done chan struct{}
	once sync.Once
}

func (c *client) drop() { c.once.Do(func() { close(c.done) }) }

// Subscription is one viewer of a showtime stream.
type Subscription struct {
	// Events delivers seat changes of the subscribed showtime only.
	Events <-chan SeatEvent
	// Done closes when the hub dropped this client (slow or shutdown).
	Done  <-chan struct{}
	close func()
}

// Close detaches the subscription; safe to call more than once.
func (s *Subscription) Close() { s.close() }

// Hub keeps one client set per showtime, so an event can never reach viewers
// of another showtime (R-S10).
type Hub struct {
	mu     sync.RWMutex
	shows  map[string]map[*client]struct{}
	closed bool
}

// NewHub creates the realtime hub.
func NewHub() *Hub {
	return &Hub{shows: make(map[string]map[*client]struct{})}
}

// Subscribe attaches a viewer to a showtime.
func (h *Hub) Subscribe(showtimeID string) *Subscription {
	c := &client{ch: make(chan SeatEvent, clientBuffer), done: make(chan struct{})}
	h.mu.Lock()
	if h.closed {
		h.mu.Unlock()
		c.drop()
		return &Subscription{Events: c.ch, Done: c.done, close: func() {}}
	}
	clients := h.shows[showtimeID]
	if clients == nil {
		clients = make(map[*client]struct{})
		h.shows[showtimeID] = clients
	}
	clients[c] = struct{}{}
	h.mu.Unlock()

	return &Subscription{Events: c.ch, Done: c.done, close: func() { h.remove(showtimeID, c) }}
}

func (h *Hub) remove(showtimeID string, c *client) {
	h.mu.Lock()
	if clients, ok := h.shows[showtimeID]; ok {
		delete(clients, c)
		if len(clients) == 0 {
			delete(h.shows, showtimeID)
		}
	}
	h.mu.Unlock()
	c.drop()
}

// Broadcast delivers an event to the viewers of exactly this showtime without
// blocking; a client whose buffer is full is dropped.
func (h *Hub) Broadcast(showtimeID string, event SeatEvent) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.shows[showtimeID] {
		select {
		case c.ch <- event:
		default:
			c.drop()
		}
	}
}

// Viewers counts the connections attached to a showtime.
func (h *Hub) Viewers(showtimeID string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.shows[showtimeID])
}

// Close drops every client so open streams end and graceful shutdown is not
// held up by long-lived connections. Later subscriptions end immediately.
func (h *Hub) Close() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.closed = true
	for id, clients := range h.shows {
		for c := range clients {
			c.drop()
		}
		delete(h.shows, id)
	}
}
