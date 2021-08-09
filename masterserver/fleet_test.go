package masterserver

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"

	"quark"
)

func TestFleet_AllocateRoom(t *testing.T) {
	fleet := NewFleet()

	addr1 := GameServerAddr{"127.0.0.1", "10000"}
	fleet.RegisterGameServer(addr1, 1)

	addr2 := GameServerAddr{"127.0.0.1", "10000"}
	fleet.RegisterGameServer(addr2, 2)

	fleet.RegisterGameServer(GameServerAddr{"127.0.0.1", "30000"}, 3)

	r1 := quark.RoomID(rand.Uint64())
	alloc1, err := fleet.AllocateRoom(r1, "")
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, addr1, alloc1)

	r2 := quark.RoomID(rand.Uint64())
	alloc2, err := fleet.AllocateRoom(r2, "")
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, addr2, alloc2)

	r3 := quark.RoomID(rand.Uint64())
	alloc3, err := fleet.AllocateRoom(r3, "")
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, addr2, alloc3)

	{
		addr, ok := fleet.LookupGameServerAddr(r1)
		assert.True(t, ok)
		assert.Equal(t, addr1, addr)
	}
	{
		addr, ok := fleet.LookupGameServerAddr(r2)
		assert.True(t, ok)
		assert.Equal(t, addr2, addr)
	}
	{
		addr, ok := fleet.LookupGameServerAddr(r3)
		assert.True(t, ok)
		assert.Equal(t, addr2, addr)
	}
}

func TestFleet_UpdateRoomStatus(t *testing.T) {
	fleet := NewFleet()

	addr1 := GameServerAddr{"127.0.0.1", "10000"}
	fleet.RegisterGameServer(addr1, 1)

	addr2 := GameServerAddr{"127.0.0.1", "20000"}
	fleet.RegisterGameServer(addr2, 2)

	addr3 := GameServerAddr{"127.0.0.1", "30000"}
	fleet.RegisterGameServer(addr3, 3)

	r1 := quark.RoomID(rand.Uint64())
	alloc1, err := fleet.AllocateRoom(r1, "")
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, addr1, alloc1)

	r2 := quark.RoomID(rand.Uint64())
	alloc2, err := fleet.AllocateRoom(r2, "")
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, addr2, alloc2)

	fleet.UpdateRoomStatus(RoomStatus{RoomID: r2, ActorCount: 2})

	r3 := quark.RoomID(rand.Uint64())
	alloc3, err := fleet.AllocateRoom(r3, "")
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, addr3, alloc3)
}
