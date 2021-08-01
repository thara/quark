package room

import (
	"math/rand"
	"sync"
)

type roomID uint64

func (r roomID) Uint64() uint64 {
	return uint64(r)
}

type roomList struct {
	rooms map[roomID]*room
	names map[string]roomID

	mux sync.RWMutex
}

func newRoomList() *roomList {
	return &roomList{
		rooms: make(map[roomID]*room),
		names: make(map[string]roomID),
	}
}

func (l *roomList) newRoom(name string) (roomID, bool) {
	id, exists := func() (roomID, bool) {
		l.mux.RLock()
		defer l.mux.RUnlock()

		if id, ok := l.names[name]; ok {
			return id, true
		} else {
			return roomID(0), false
		}
	}()
	if exists {
		return id, true
	}

	newID := roomID(rand.Uint64())
	room := newRoom()

	func() {
		l.mux.Lock()
		defer l.mux.Unlock()
		l.rooms[newID] = room
		l.names[name] = newID
	}()

	return newID, false
}

func (l *roomList) getRoom(id roomID) (*room, bool) {
	l.mux.RLock()
	defer l.mux.RUnlock()

	if r, ok := l.rooms[id]; ok {
		return r, true
	} else {
		return nil, false
	}
}
