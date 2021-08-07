package quark

import (
	"math/rand"
	"sync"

	"github.com/google/uuid"
)

type RoomID uint64

func (r RoomID) Uint64() uint64 {
	return uint64(r)
}

type RoomSet struct {
	rooms map[RoomID]*Room
	names map[string]RoomID

	mux sync.RWMutex
}

func NewRoomSet() *RoomSet {
	return &RoomSet{
		rooms: make(map[RoomID]*Room),
		names: make(map[string]RoomID),
	}
}

func (s *RoomSet) Rooms() []*Room {
	rs := make([]*Room, 0, len(s.rooms))
	for _, r := range s.rooms {
		rs = append(rs, r)
	}
	return rs
}

func (s *RoomSet) NewRoom(name string) (RoomID, bool) {
	if len(name) == 0 {
		name = uuid.Must(uuid.NewRandom()).String()
	}

	id, exists := func() (RoomID, bool) {
		s.mux.RLock()
		defer s.mux.RUnlock()

		if id, ok := s.names[name]; ok {
			return id, true
		} else {
			return RoomID(0), false
		}
	}()
	if exists {
		return id, true
	}

	newID := RoomID(rand.Uint64())
	room := NewRoom()

	func() {
		s.mux.Lock()
		defer s.mux.Unlock()
		s.rooms[newID] = room
		s.names[name] = newID
	}()

	return newID, false
}

func (s *RoomSet) GetRoom(id RoomID) (*Room, bool) {
	s.mux.RLock()
	defer s.mux.RUnlock()

	if r, ok := s.rooms[id]; ok {
		return r, true
	} else {
		return nil, false
	}
}

func (s *RoomSet) JoinRoom(roomID RoomID, a *Actor) bool {
	r, ok := s.GetRoom(roomID)
	if !ok {
		return false
	}
	a.JoinTo(r)
	return true
}
