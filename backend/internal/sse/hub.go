// Package sse streams realtime seat changes per showtime.
package sse

import (
	"errors"
	"sync"
)

// SeatUpdate.ID is the showtime_seat id.
type SeatUpdate struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

type SeatEvent struct {
	ShowtimeID string       `json:"showtime_id"`
	Seats      []SeatUpdate `json:"seats"`
}

// A client that falls this far behind is dropped instead of slowing everyone down.
const clientBuffer = 64

const (
	DefaultMaxStreamsPerUser = 5
	DefaultMaxStreams        = 2000
)

var ErrTooManyStreams = errors.New("too many realtime streams")

type client struct {
	user string
	ch   chan SeatEvent
	done chan struct{}
	once sync.Once
}

func (c *client) drop() { c.once.Do(func() { close(c.done) }) }

type Subscription struct {
	Events <-chan SeatEvent
	// Done closes when the hub drops this client (too slow, or shutdown).
	Done  <-chan struct{}
	close func()
}

// Close is safe to call more than once.
func (s *Subscription) Close() { s.close() }

// Hub keeps one client set per showtime so an event never reaches viewers of another showtime.
type Hub struct {
	// Set the stream limits before the hub is used.
	MaxStreamsPerUser int
	MaxStreams        int

	mu      sync.RWMutex
	shows   map[string]map[*client]struct{}
	perUser map[string]int
	total   int
	closed  bool
}

func NewHub() *Hub {
	return &Hub{
		MaxStreamsPerUser: DefaultMaxStreamsPerUser,
		MaxStreams:        DefaultMaxStreams,
		shows:             make(map[string]map[*client]struct{}),
		perUser:           make(map[string]int),
	}
}

// Subscribe returns ErrTooManyStreams when the user or the server has too many streams open.
func (h *Hub) Subscribe(showtimeID, userID string) (*Subscription, error) {
	c := &client{user: userID, ch: make(chan SeatEvent, clientBuffer), done: make(chan struct{})}
	h.mu.Lock()
	if h.closed {
		h.mu.Unlock()
		c.drop()
		return &Subscription{Events: c.ch, Done: c.done, close: func() {}}, nil
	}
	if h.total >= h.MaxStreams || h.perUser[userID] >= h.MaxStreamsPerUser {
		h.mu.Unlock()
		return nil, ErrTooManyStreams
	}
	clients := h.shows[showtimeID]
	if clients == nil {
		clients = make(map[*client]struct{})
		h.shows[showtimeID] = clients
	}
	clients[c] = struct{}{}
	h.perUser[userID]++
	h.total++
	h.mu.Unlock()

	var once sync.Once
	return &Subscription{Events: c.ch, Done: c.done, close: func() {
		once.Do(func() { h.remove(showtimeID, c) })
	}}, nil
}

func (h *Hub) remove(showtimeID string, c *client) {
	h.mu.Lock()
	if clients, ok := h.shows[showtimeID]; ok {
		if _, attached := clients[c]; attached {
			delete(clients, c)
			h.release(c.user)
		}
		if len(clients) == 0 {
			delete(h.shows, showtimeID)
		}
	}
	h.mu.Unlock()
	c.drop()
}

// The caller must hold h.mu.
func (h *Hub) release(user string) {
	h.total--
	if h.perUser[user] <= 1 {
		delete(h.perUser, user)
	} else {
		h.perUser[user]--
	}
}

// Broadcast never blocks: a client whose buffer is full is dropped.
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

func (h *Hub) Viewers(showtimeID string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.shows[showtimeID])
}

// Close drops every client so long-lived streams don't hold up graceful shutdown.
// Later subscriptions end immediately.
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
	h.perUser = make(map[string]int)
	h.total = 0
}
