package gameserver

type RoomEvent interface {
	EventType() RoomEventType
}

type RoomEventType uint64

const (
	_ RoomEventType = iota
	OnJoinRoom
	OnLeaveRoom
)

type JoinRoomEvent struct {
	ActorList []ActorID
	NewActor  ActorID
}

func (e *JoinRoomEvent) EventType() RoomEventType {
	return OnJoinRoom
}

type LeaveRoomEvent struct {
	ActorList    []ActorID
	RemovedActor ActorID
}

func (e *LeaveRoomEvent) EventType() RoomEventType {
	return OnLeaveRoom
}
