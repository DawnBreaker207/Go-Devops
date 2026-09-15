// Package notify sends emails; this build only has a mock mailer that writes .html files to an outbox.
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

type Message struct {
	To      string
	Subject string
	HTML    string
}

type Mailer interface {
	Send(ctx context.Context, msg Message) error
}

type MockMailer struct {
	outboxDir string

	mu   sync.Mutex
	sent []Message
	fail error
}

// NewMockMailer keeps messages in memory only when outboxDir is empty.
func NewMockMailer(outboxDir string) *MockMailer {
	return &MockMailer{outboxDir: outboxDir}
}

// Recipient addresses never reach logs or file names.
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

// Sent returns a copy of the sent messages.
func (m *MockMailer) Sent() []Message {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]Message(nil), m.sent...)
}

// FailWith makes every Send fail with err until called again with nil.
func (m *MockMailer) FailWith(err error) {
	m.mu.Lock()
	m.fail = err
	m.mu.Unlock()
}
