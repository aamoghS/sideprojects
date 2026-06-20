package store

import (
	"sync"
	"time"
)

type RoomChat struct {
	Room      string
	Sender    string
	Text      string
	Timestamp time.Time
}

type PresenceKind int

const (
	PresenceJoin PresenceKind = iota + 1
	PresenceLeave
)

type RoomPresence struct {
	Room             string
	Kind             PresenceKind
	Participant      string
	ParticipantCount int
}

type roomParticipant struct {
	nickname string
	send     chan RoomFrame
}

type RoomFrame struct {
	Chat     *RoomChat
	Presence *RoomPresence
}

type RoomStore struct {
	mu    sync.RWMutex
	rooms map[string]map[*roomParticipant]struct{}
}

func NewRoomStore() *RoomStore {
	return &RoomStore{rooms: make(map[string]map[*roomParticipant]struct{})}
}

func (rs *RoomStore) Join(room, nickname string) (<-chan RoomFrame, func()) {
	p := &roomParticipant{
		nickname: nickname,
		send:     make(chan RoomFrame, 32),
	}

	rs.mu.Lock()
	participants := rs.roomLocked(room)
	participants[p] = struct{}{}
	count := len(participants)
	rs.broadcastLocked(room, RoomFrame{
		Presence: &RoomPresence{
			Room:             room,
			Kind:             PresenceJoin,
			Participant:      nickname,
			ParticipantCount: count,
		},
	}, p)
	rs.mu.Unlock()

	leave := func() {
		rs.mu.Lock()
		defer rs.mu.Unlock()
		participants, ok := rs.rooms[room]
		if !ok {
			return
		}
		delete(participants, p)
		count := len(participants)
		rs.broadcastLocked(room, RoomFrame{
			Presence: &RoomPresence{
				Room:             room,
				Kind:             PresenceLeave,
				Participant:      nickname,
				ParticipantCount: count,
			},
		}, nil)
		if count == 0 {
			delete(rs.rooms, room)
		}
		close(p.send)
	}
	return p.send, leave
}

func (rs *RoomStore) Chat(room, sender, text string) {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	participants, ok := rs.rooms[room]
	if !ok || len(participants) == 0 {
		return
	}
	rs.broadcastLocked(room, RoomFrame{
		Chat: &RoomChat{
			Room:      room,
			Sender:    sender,
			Text:      text,
			Timestamp: time.Now().UTC(),
		},
	}, nil)
}

func (rs *RoomStore) ParticipantCount(room string) int {
	rs.mu.RLock()
	defer rs.mu.RUnlock()
	return len(rs.rooms[room])
}

func (rs *RoomStore) roomLocked(room string) map[*roomParticipant]struct{} {
	participants, ok := rs.rooms[room]
	if !ok {
		participants = make(map[*roomParticipant]struct{})
		rs.rooms[room] = participants
	}
	return participants
}

func (rs *RoomStore) broadcastLocked(room string, frame RoomFrame, skip *roomParticipant) {
	participants := rs.rooms[room]
	for p := range participants {
		if p == skip {
			continue
		}
		select {
		case p.send <- frame:
		default:
		}
	}
}
