package masterserver

import (
	"sync"

	"quark"
)

type GameServer struct {
	id      GameServerID
	addr    GameServerAddr
	rooms   map[quark.RoomID]*RoomStatus
	nActors uint
	roomCap uint
	mux     sync.RWMutex
}

func newGameServer(id GameServerID, addr GameServerAddr, roomCap uint) *GameServer {
	return &GameServer{id: id, addr: addr, rooms: make(map[quark.RoomID]*RoomStatus), roomCap: roomCap}
}

func (g *GameServer) Cap() uint {
	return g.roomCap - uint(len(g.rooms))
}

func (g *GameServer) HasCapacity() bool {
	g.mux.RLock()
	defer g.mux.RUnlock()
	return len(g.rooms) < int(g.roomCap)
}

func (g *GameServer) AddRoom(roomID quark.RoomID) error {
	g.mux.Lock()
	defer g.mux.Unlock()

	_, ok := g.rooms[roomID]
	if ok {
		return ErrRoomAlreadyAllocated
	}
	g.rooms[roomID] = &RoomStatus{RoomID: roomID}
	return nil
}

func (g *GameServer) UpdateRoomStatus(status RoomStatus) error {
	g.mux.Lock()
	defer g.mux.Unlock()

	_, ok := g.rooms[status.RoomID]
	if !ok {
		return ErrRoomStatusNotFound
	}
	g.rooms[status.RoomID] = &status

	var n uint = 0
	for _, s := range g.rooms {
		n += s.ActorCount
	}
	g.nActors = n
	return nil
}
