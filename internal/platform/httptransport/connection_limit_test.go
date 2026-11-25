package httptransport

import (
	"context"
	"net"
	"testing"
	"time"
)

func TestConnectionLimitWaiterUnblocksWhenListenerCloses(t *testing.T) {
	var listenConfig net.ListenConfig
	base, err := listenConfig.Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	slots := make(chan struct{}, 1)
	slots <- struct{}{}
	limited := &connectionLimitedListener{Listener: base, slots: slots, done: make(chan struct{})}
	result := make(chan error, 1)
	go func() {
		_, acceptErr := limited.Accept()
		result <- acceptErr
	}()
	if err := limited.Close(); err != nil {
		t.Fatalf("close limited listener: %v", err)
	}
	select {
	case acceptErr := <-result:
		if acceptErr == nil {
			t.Fatal("blocked Accept returned nil after close")
		}
	case <-time.After(time.Second):
		t.Fatal("connection-limit waiter remained blocked after close")
	}
}
