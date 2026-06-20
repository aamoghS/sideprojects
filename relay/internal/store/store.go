package store

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

const defaultBufferSize = 256

type Event struct {
	ID        string
	Topic     string
	Payload   []byte
	Timestamp time.Time
	Metadata  map[string]string
	Offset    int64
}

type ringBuffer struct {
	size   int
	slots  []Event
	head   int
	count  int
	nextID int64
}

func newRingBuffer(size int) *ringBuffer {
	if size <= 0 {
		size = defaultBufferSize
	}
	return &ringBuffer{
		size:  size,
		slots: make([]Event, size),
	}
}

func (r *ringBuffer) append(ev Event) int64 {
	offset := r.nextID
	r.nextID++
	ev.Offset = offset
	r.slots[r.head] = ev
	r.head = (r.head + 1) % r.size
	if r.count < r.size {
		r.count++
	}
	return offset
}

func (r *ringBuffer) replay(from int64) []Event {
	if r.count == 0 || from >= r.nextID {
		return nil
	}
	oldest := r.nextID - int64(r.count)
	if from < oldest {
		from = oldest
	}
	out := make([]Event, 0, r.nextID-from)
	oldestIdx := (r.head - r.count + r.size) % r.size
	for off := from; off < r.nextID; off++ {
		idx := (oldestIdx + int(off-oldest)) % r.size
		out = append(out, r.slots[idx])
	}
	return out
}

type subscriber struct {
	ch     chan Event
	topics map[string]struct{}
}

type topicState struct {
	buffer *ringBuffer
	subs   map[*subscriber]struct{}
}

type Store struct {
	mu     sync.RWMutex
	topics map[string]*topicState
}

func New() *Store {
	return &Store{topics: make(map[string]*topicState)}
}

func (s *Store) Publish(topic string, payload []byte, metadata map[string]string) (Event, int) {
	ev := Event{
		ID:        newEventID(),
		Topic:     topic,
		Payload:   append([]byte(nil), payload...),
		Timestamp: time.Now().UTC(),
		Metadata:  cloneMeta(metadata),
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	ts := s.topicLocked(topic)
	offset := ts.buffer.append(ev)
	ev.Offset = offset

	subs := make([]*subscriber, 0, len(ts.subs))
	for sub := range ts.subs {
		subs = append(subs, sub)
	}

	for _, sub := range subs {
		select {
		case sub.ch <- ev:
		default:
		}
	}
	return ev, len(subs)
}

func (s *Store) Subscribe(topics []string, fromOffset int64) (<-chan Event, func()) {
	sub := &subscriber{
		ch:     make(chan Event, 64),
		topics: make(map[string]struct{}, len(topics)),
	}
	for _, t := range topics {
		sub.topics[t] = struct{}{}
	}

	replay := make([]Event, 0)
	s.mu.Lock()
	for _, t := range topics {
		ts := s.topicLocked(t)
		ts.subs[sub] = struct{}{}
		if fromOffset > 0 {
			replay = append(replay, ts.buffer.replay(fromOffset)...)
		}
	}
	s.mu.Unlock()

	go func() {
		for _, ev := range replay {
			sub.ch <- ev
		}
	}()

	unsub := func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		for t := range sub.topics {
			if ts, ok := s.topics[t]; ok {
				delete(ts.subs, sub)
			}
		}
		close(sub.ch)
	}
	return sub.ch, unsub
}

func (s *Store) SubscriberCount(topic string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ts, ok := s.topics[topic]
	if !ok {
		return 0
	}
	return len(ts.subs)
}

func (s *Store) topicLocked(topic string) *topicState {
	ts, ok := s.topics[topic]
	if !ok {
		ts = &topicState{
			buffer: newRingBuffer(defaultBufferSize),
			subs:   make(map[*subscriber]struct{}),
		}
		s.topics[topic] = ts
	}
	return ts
}

func newEventID() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

func cloneMeta(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
