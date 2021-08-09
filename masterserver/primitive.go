package masterserver

import "quark"

type RoomStatus struct {
	RoomID     quark.RoomID
	RoomName   string
	ActorCount uint
}

type RoomAllocatedEvent struct {
	GameServer GameServerAddr
	Room       RoomStatus
}

type GameServerID string

type GameServerAddr struct {
	Addr string
	Port string
}
