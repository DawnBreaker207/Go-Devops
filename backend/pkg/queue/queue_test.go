package queue

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
)

func testURL() string {
	if v := os.Getenv("QUEUE_TEST_URL"); v != "" {
		return v
	}
	return "amqp://guest:guest@localhost:5672/"
}

func dialOrSkip(t *testing.T, url string) *Client {
	t.Helper()
	c, err := Dial(url)
	if err != nil {
		t.Skipf("rabbitmq unavailable at %s: %v", url, err)
	}
	t.Cleanup(func() { _ = c.Close() })
	return c
}

func tempQueue(t *testing.T, url string) string {
	name := "test." + uuid.NewString()[:8]
	t.Cleanup(func() { deleteQueue(url, name) })
	return name
}

func deleteQueue(url, name string) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return
	}
	defer conn.Close()
	ch, err := conn.Channel()
	if err != nil {
		return
	}
	defer ch.Close()
	_, _ = ch.QueueDelete(name, false, false, false)
}

// consumeN consumes until n messages arrived or the timeout passed.
func consumeN(c *Client, queueName string, n int, timeout time.Duration) []string {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	var mu sync.Mutex
	var got []string
	done := make(chan struct{})
	go func() {
		_ = c.Consume(ctx, queueName, func(_ context.Context, body []byte) error {
			mu.Lock()
			defer mu.Unlock()
			got = append(got, string(body))
			if len(got) == n {
				close(done)
			}
			return nil
		})
	}()
	select {
	case <-done:
	case <-ctx.Done():
	}
	mu.Lock()
	defer mu.Unlock()
	out := append([]string(nil), got...)
	sort.Strings(out)
	return out
}

func mustPublish(t *testing.T, c *Client, queueName, body string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := c.Publish(ctx, queueName, []byte(body)); err != nil {
		t.Fatalf("publish %s: %v", body, err)
	}
}

func TestPublishConsume_Durable(t *testing.T) {
	url := testURL()
	c := dialOrSkip(t, url)
	q := tempQueue(t, url)
	for i := 1; i <= 3; i++ {
		mustPublish(t, c, q, fmt.Sprintf("m%d", i))
	}
	if got := consumeN(c, q, 3, 20*time.Second); len(got) != 3 {
		t.Fatalf("consumed %v, want 3 messages", got)
	}
}

// The broker dropping the connection must not break the next publish.
func TestPublish_RedialsAfterConnectionDrop(t *testing.T) {
	url := testURL()
	c := dialOrSkip(t, url)
	q := tempQueue(t, url)
	mustPublish(t, c, q, "before")

	c.mu.Lock()
	_ = c.conn.Close()
	c.mu.Unlock()

	mustPublish(t, c, q, "after")
	if got := consumeN(c, q, 2, 20*time.Second); len(got) != 2 || got[0] != "after" || got[1] != "before" {
		t.Fatalf("consumed %v, want [after before]", got)
	}
}

func waitForBroker(t *testing.T, url string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if conn, err := amqp.Dial(url); err == nil {
			_ = conn.Close()
			return
		}
		time.Sleep(time.Second)
	}
	t.Fatalf("broker at %s not reachable after %v", url, timeout)
}

func restartBroker(t *testing.T, container, url string) {
	t.Helper()
	if out, err := exec.Command("docker", "restart", container).CombinedOutput(); err != nil {
		t.Fatalf("docker restart %s: %v %s", container, err, out)
	}
	waitForBroker(t, url, 90*time.Second)
}

// T26 drill: the broker restarts mid-way. Messages already queued survive, the
// same client publishes again, and a running consumer reconnects by itself.
// It restarts a container, so it only runs against a disposable broker:
//
//	docker run -d --name cp-rabbitmq-chaos -p 5673:5672 rabbitmq:3.13-management-alpine
//	QUEUE_CHAOS_CONTAINER=cp-rabbitmq-chaos QUEUE_CHAOS_URL=amqp://guest:guest@localhost:5673/ \
//	  go test ./pkg/queue -run TestBrokerRestart -v
func TestBrokerRestart_NoMessageLost(t *testing.T) {
	container, url := os.Getenv("QUEUE_CHAOS_CONTAINER"), os.Getenv("QUEUE_CHAOS_URL")
	if container == "" || url == "" {
		t.Skip("set QUEUE_CHAOS_CONTAINER and QUEUE_CHAOS_URL to run the broker restart drill")
	}
	waitForBroker(t, url, 90*time.Second)
	c := dialOrSkip(t, url)
	q := tempQueue(t, url)

	for i := 1; i <= 3; i++ {
		mustPublish(t, c, q, fmt.Sprintf("queued-%d", i))
	}
	restartBroker(t, container, url)
	mustPublish(t, c, q, "published-after-restart-1")
	mustPublish(t, c, q, "published-after-restart-2")
	if got := consumeN(c, q, 5, 60*time.Second); len(got) != 5 {
		t.Fatalf("after restart consumed %v, want 5 messages", got)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	received := make(chan string, 8)
	go func() {
		_ = c.Consume(ctx, q, func(_ context.Context, body []byte) error {
			received <- string(body)
			return nil
		})
	}()
	time.Sleep(2 * time.Second)
	restartBroker(t, container, url)
	mustPublish(t, c, q, "while-consumer-reconnects")
	select {
	case body := <-received:
		if body != "while-consumer-reconnects" {
			t.Fatalf("consumer got %q", body)
		}
	case <-time.After(60 * time.Second):
		t.Fatal("consumer did not reconnect after the broker restart")
	}
}
