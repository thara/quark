package quark

import (
	"errors"
	"fmt"
	"sort"
	"sync"

	"github.com/google/uuid"
)

var (
	ErrNotEnoughGameServers = errors.New("not enough game servers")
	ErrRoomAlreadyAllocated = errors.New("room already allocated")
	ErrRoomStatusNotFound   = errors.New("room status not found")
)

type RoomStatus struct {
	roomID     RoomID
	actorCount int
}

type GameServerID string

type GameServerAddr struct {
	addr string
	port string
}

type GameServer struct {
	id      GameServerID
	addr    GameServerAddr
	rooms   map[RoomID]*RoomStatus
	nActors int
	roomCap int
	mux     sync.RWMutex
}

func newGameServer(id GameServerID, addr GameServerAddr, roomCap int) *GameServer {
	return &GameServer{id: id, addr: addr, rooms: make(map[RoomID]*RoomStatus), roomCap: roomCap}
}

func (g *GameServer) Cap() int {
	return g.roomCap - len(g.rooms)
}

func (g *GameServer) HasCapacity() bool {
	g.mux.RLock()
	defer g.mux.RUnlock()
	return len(g.rooms) < g.roomCap
}

func (g *GameServer) AddRoom(roomID RoomID) error {
	g.mux.Lock()
	defer g.mux.Unlock()

	_, ok := g.rooms[roomID]
	if ok {
		return ErrRoomAlreadyAllocated
	}
	g.rooms[roomID] = &RoomStatus{roomID: roomID}
	return nil
}

func (g *GameServer) UpdateRoomStatus(status RoomStatus) error {
	g.mux.Lock()
	defer g.mux.Unlock()

	_, ok := g.rooms[status.roomID]
	if !ok {
		return ErrRoomStatusNotFound
	}
	g.rooms[status.roomID] = &status

	n := 0
	for _, s := range g.rooms {
		n += s.actorCount
	}
	g.nActors = n
	return nil
}

type Fleet struct {
	rs  map[RoomID]*RoomStatus
	rg  map[RoomID]*GameServer
	g   []*GameServer
	mux sync.RWMutex
}

func NewFleet() *Fleet {
	return &Fleet{
		rs: make(map[RoomID]*RoomStatus),
		rg: make(map[RoomID]*GameServer),
		g:  make([]*GameServer, 0),
	}
}

func (f *Fleet) RegisterGameServer(addr GameServerAddr, cap int) GameServerID {
	f.mux.Lock()
	defer f.mux.Unlock()

	id := GameServerID(uuid.Must(uuid.NewRandom()).String())
	fmt.Println("GameServerID", id)
	gs := newGameServer(id, addr, cap)
	f.g = append(f.g, gs)
	return id
}

func (f *Fleet) AllocateRoom(roomID RoomID) (GameServerAddr, error) {
	err := func() error {
		f.mux.RLock()
		defer f.mux.RUnlock()

		if len(f.g) == 0 {
			return ErrNotEnoughGameServers
		}
		_, ok := f.rg[roomID]
		if ok {
			return ErrRoomAlreadyAllocated
		}
		return nil
	}()
	if err != nil {
		return GameServerAddr{}, err
	}
	f.mux.Lock()
	defer f.mux.Unlock()

	var lookup func(gs []*GameServer) *GameServer
	lookup = func(gs []*GameServer) *GameServer {
		if len(gs) == 0 {
			return nil
		}
		g := gs[0]
		if g.HasCapacity() {
			return g
		}
		return lookup(gs[1:])
	}

	g := lookup(f.g)
	if g == nil {
		return GameServerAddr{}, ErrNotEnoughGameServers
	}

	err = g.AddRoom(roomID)
	if err != nil {
		return GameServerAddr{}, err
	}

	f.rg[roomID] = g
	f.rs[roomID] = &RoomStatus{roomID: roomID, actorCount: 0}
	return g.addr, nil
}

func (f *Fleet) LookupGameServerAddr(roomID RoomID) (GameServerAddr, bool) {
	f.mux.RLock()
	defer f.mux.RUnlock()
	g, ok := f.rg[roomID]
	return g.addr, ok
}

func (f *Fleet) UpdateRoomStatus(status RoomStatus) error {
	f.mux.Lock()
	defer f.mux.Unlock()
	roomID := status.roomID

	_, ok := f.rs[roomID]
	if !ok {
		return ErrRoomStatusNotFound
	}
	f.rs[roomID] = &status

	gs, ok := f.rg[roomID]
	if !ok {
		return errors.New("game server not found")
	}
	gs.UpdateRoomStatus(status)

	sort.SliceStable(f.g, func(i, j int) bool {
		return f.g[i].Cap() > f.g[j].Cap()
	})
	return nil
}
