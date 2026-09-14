// Package notify sends customer notifications. This build ships a mock mailer
// only: messages are kept in memory and written as .html files to an outbox
// directory so they can be opened in a browser (no SMTP, no Mailtrap).
package notify

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/Cinema-Project-Juann/BackEnd-CP/pkg/logger"
)

// Message is one email.
type Message struct {
	To      string
	Subject string
	HTML    string
}

// Mailer delivers emails.
type Mailer interface {
	Send(ctx context.Context, msg Message) error
}

// MockMailer records messages instead of sending them.
type MockMailer struct {
	outboxDir string

	mu   sync.Mutex
	sent []Message
	fail error
}

// NewMockMailer builds a mock mailer; an empty outboxDir keeps messages in
// memory only.
func NewMockMailer(outboxDir string) *MockMailer {
	return &MockMailer{outboxDir: outboxDir}
}

// Send stores the message, or returns the configured failure. Recipient
// addresses never reach logs or file names (NFR-LEG-01).
func (m *MockMailer) Send(_ context.Context, msg Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.fail != nil {
		return m.fail
	}
	if m.outboxDir != "" {
		if err := os.MkdirAll(m.outboxDir, 0o755); err != nil {
			return fmt.Errorf("create mail outbox: %w", err)
		}
		name := fmt.Sprintf("%s_%s.html", time.Now().Format("20060102-150405.000"), uuid.NewString()[:8])
		path := filepath.Join(m.outboxDir, name)
		if err := os.WriteFile(path, []byte(msg.HTML), 0o644); err != nil {
			return fmt.Errorf("write mail outbox: %w", err)
		}
		logger.Info("mock email written", logger.String("file", path))
	} else {
		logger.Info("mock email sent")
	}
	m.sent = append(m.sent, msg)
	return nil
}

// Sent returns a copy of every delivered message.
func (m *MockMailer) Sent() []Message {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]Message(nil), m.sent...)
}

// FailWith makes every Send fail with err until called again with nil
// (simulates a broken mail server, E-ML1).
func (m *MockMailer) FailWith(err error) {
	m.mu.Lock()
	m.fail = err
	m.mu.Unlock()
}
