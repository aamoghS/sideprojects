package store

import (
	"testing"
)

func TestRingBufferReplay(t *testing.T) {
	tests := []struct {
		name     string
		writes   []string
		from     int64
		wantLen  int
		wantText []string
	}{
		{
			name:    "empty buffer",
			from:    0,
			wantLen: 0,
		},
		{
			name:     "replay all",
			writes:   []string{"a", "b", "c"},
			from:     0,
			wantLen:  3,
			wantText: []string{"a", "b", "c"},
		},
		{
			name:     "replay from offset",
			writes:   []string{"a", "b", "c"},
			from:     1,
			wantLen:  2,
			wantText: []string{"b", "c"},
		},
		{
			name:    "from beyond head",
			writes:  []string{"a"},
			from:    99,
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rb := newRingBuffer(8)
			for _, w := range tt.writes {
				rb.append(Event{Payload: []byte(w)})
			}
			got := rb.replay(tt.from)
			if len(got) != tt.wantLen {
				t.Fatalf("replay len = %d, want %d", len(got), tt.wantLen)
			}
			for i, want := range tt.wantText {
				if string(got[i].Payload) != want {
					t.Fatalf("event[%d] = %q, want %q", i, got[i].Payload, want)
				}
			}
		})
	}
}

func TestPublishDeliveredToSubscriber(t *testing.T) {
	tests := []struct {
		name  string
		topic string
	}{
		{name: "logs topic", topic: "logs"},
		{name: "metrics topic", topic: "metrics"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := New()
			ch, unsub := s.Subscribe([]string{tt.topic}, 0)
			defer unsub()

			ev, count := s.Publish(tt.topic, []byte("hello"), nil)
			if count != 1 {
				t.Fatalf("subscriber count = %d, want 1", count)
			}

			select {
			case got := <-ch:
				if got.ID != ev.ID {
					t.Fatalf("event id mismatch: got %q want %q", got.ID, ev.ID)
				}
				if string(got.Payload) != "hello" {
					t.Fatalf("payload = %q, want hello", got.Payload)
				}
			default:
				t.Fatal("expected delivered event")
			}
		})
	}
}

func TestReplayOnSubscribe(t *testing.T) {
	s := New()
	_, _ = s.Publish("dev", []byte("one"), nil)
	_, _ = s.Publish("dev", []byte("two"), nil)

	ch, unsub := s.Subscribe([]string{"dev"}, 1)
	defer unsub()

	got := <-ch
	if string(got.Payload) != "two" {
		t.Fatalf("replay = %q, want two", got.Payload)
	}
}

func TestSubscriberCount(t *testing.T) {
	s := New()
	if got := s.SubscriberCount("nope"); got != 0 {
		t.Fatalf("count = %d, want 0", got)
	}
	_, unsub := s.Subscribe([]string{"live"}, 0)
	defer unsub()
	if got := s.SubscriberCount("live"); got != 1 {
		t.Fatalf("count = %d, want 1", got)
	}
}
