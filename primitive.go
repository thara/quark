package quark

import "math/rand"

type RoomID uint64

func (r RoomID) Uint64() uint64 {
	return uint64(r)
}

func NewRoomID() RoomID {
	return RoomID(rand.Uint64())
}
