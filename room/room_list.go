package room

import (
	"math/rand"
	"sync"
)

type RoomID uint64

func (r RoomID) Uint64() uint64 {
	return uint64(r)
}

type RoomList struct {
	rooms map[RoomID]*Room
	names map[string]RoomID

	mux sync.RWMutex
}

func NewRoomList() *RoomList {
	return &RoomList{
		rooms: make(map[RoomID]*Room),
		names: make(map[string]RoomID),
	}
}

func (l *RoomList) Rooms() []*Room {
	s := make([]*Room, len(l.rooms))
	for _, r := range l.rooms {
		s = append(s, r)
	}
	return s
}

func (l *RoomList) NewRoom(name string) (RoomID, bool) {
	id, exists := func() (RoomID, bool) {
		l.mux.RLock()
		defer l.mux.RUnlock()

		if id, ok := l.names[name]; ok {
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
		l.mux.Lock()
		defer l.mux.Unlock()
		l.rooms[newID] = room
		l.names[name] = newID
	}()

	return newID, false
}

func (l *RoomList) GetRoom(id RoomID) (*Room, bool) {
	l.mux.RLock()
	defer l.mux.RUnlock()

	if r, ok := l.rooms[id]; ok {
		return r, true
	} else {
		return nil, false
	}
}
