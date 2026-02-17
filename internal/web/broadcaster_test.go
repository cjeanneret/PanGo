package web

import (
	"encoding/json"
	"testing"
	"time"
)

func TestBroadcaster_SubscribeAndReceive(t *testing.T) {
	b := NewStatusBroadcaster()
	ch, unsub := b.Subscribe()
	defer unsub()

	b.Broadcast("info", "hello")

	select {
	case msg := <-ch:
		var evt StatusEvent
		if err := json.Unmarshal([]byte(msg), &evt); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if evt.Msg != "hello" {
			t.Errorf("msg = %q, want \"hello\"", evt.Msg)
		}
		if evt.Level != "info" {
			t.Errorf("level = %q, want \"info\"", evt.Level)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for broadcast")
	}
}

func TestBroadcaster_MultipleSubscribers(t *testing.T) {
	b := NewStatusBroadcaster()
	ch1, unsub1 := b.Subscribe()
	defer unsub1()
	ch2, unsub2 := b.Subscribe()
	defer unsub2()

	b.Broadcast("info", "multi")

	for i, ch := range []<-chan string{ch1, ch2} {
		select {
		case msg := <-ch:
			var evt StatusEvent
			if err := json.Unmarshal([]byte(msg), &evt); err != nil {
				t.Fatalf("subscriber %d: unmarshal: %v", i, err)
			}
			if evt.Msg != "multi" {
				t.Errorf("subscriber %d: msg = %q, want \"multi\"", i, evt.Msg)
			}
		case <-time.After(time.Second):
			t.Fatalf("subscriber %d: timeout", i)
		}
	}
}

func TestBroadcaster_UnsubscribeClosesChannel(t *testing.T) {
	b := NewStatusBroadcaster()
	ch, unsub := b.Subscribe()
	unsub()

	// Channel should be closed after unsubscribe
	_, ok := <-ch
	if ok {
		t.Error("expected channel to be closed after unsubscribe")
	}
}

func TestBroadcaster_FullChannelDropsMessage(t *testing.T) {
	b := NewStatusBroadcaster()
	ch, unsub := b.Subscribe()
	defer unsub()

	// Fill the channel buffer (64 messages)
	for i := 0; i < 64; i++ {
		b.Broadcast("info", "fill")
	}

	// This should not panic or block — message should be silently dropped
	b.Broadcast("info", "overflow")

	// Drain and count messages
	count := 0
	for {
		select {
		case <-ch:
			count++
		default:
			goto done
		}
	}
done:
	if count != 64 {
		t.Errorf("expected 64 buffered messages, got %d", count)
	}
}

func TestBroadcaster_AfterUnsubscribeBroadcastDoesNotPanic(t *testing.T) {
	b := NewStatusBroadcaster()
	_, unsub := b.Subscribe()
	unsub()

	// Broadcasting after unsubscribe should not panic
	b.Broadcast("info", "after unsub")
}

func TestBroadcaster_BroadcastMsg(t *testing.T) {
	b := NewStatusBroadcaster()
	ch, unsub := b.Subscribe()
	defer unsub()

	b.BroadcastMsg("convenience")

	select {
	case msg := <-ch:
		var evt StatusEvent
		if err := json.Unmarshal([]byte(msg), &evt); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if evt.Level != "info" {
			t.Errorf("level = %q, want \"info\"", evt.Level)
		}
		if evt.Msg != "convenience" {
			t.Errorf("msg = %q, want \"convenience\"", evt.Msg)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout")
	}
}

func TestBroadcastWriter_Write(t *testing.T) {
	b := NewStatusBroadcaster()
	ch, unsub := b.Subscribe()
	defer unsub()

	w := BroadcastWriter(b)
	n, err := w.Write([]byte("  trimmed message  \n"))
	if err != nil {
		t.Fatalf("Write error: %v", err)
	}
	if n != len("  trimmed message  \n") {
		t.Errorf("n = %d, want %d", n, len("  trimmed message  \n"))
	}

	select {
	case msg := <-ch:
		var evt StatusEvent
		if err := json.Unmarshal([]byte(msg), &evt); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if evt.Msg != "trimmed message" {
			t.Errorf("msg = %q, want \"trimmed message\"", evt.Msg)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout")
	}
}

func TestBroadcastWriter_EmptyWriteIgnored(t *testing.T) {
	b := NewStatusBroadcaster()
	ch, unsub := b.Subscribe()
	defer unsub()

	w := BroadcastWriter(b)
	w.Write([]byte("   \n"))

	select {
	case <-ch:
		t.Error("expected no message for whitespace-only write")
	case <-time.After(50 * time.Millisecond):
		// expected: no message
	}
}

func TestBroadcaster_EventHasTimestamp(t *testing.T) {
	b := NewStatusBroadcaster()
	ch, unsub := b.Subscribe()
	defer unsub()

	b.Broadcast("info", "timestamped")

	select {
	case msg := <-ch:
		var evt StatusEvent
		if err := json.Unmarshal([]byte(msg), &evt); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if evt.Time == "" {
			t.Error("event should have a timestamp")
		}
	case <-time.After(time.Second):
		t.Fatal("timeout")
	}
}
