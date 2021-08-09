package gameserver

import "github.com/google/uuid"

type ActorID string

func NewActorID() ActorID {
	return ActorID(uuid.Must(uuid.NewRandom()).String())
}

func (a ActorID) String() string {
	return string(a)
}

type Message interface{}

type ActorMessage struct {
	Sender  ActorID
	Code    uint32
	Payload []byte
}

type RoomEntry struct {
	id ActorID
	r  *Room
	s  chan Message
}

func (e *RoomEntry) Subscription() <-chan Message {
	return e.s
}

func (e *RoomEntry) Send(m ActorMessage) {
	e.r.messages <- m
}

func (e *RoomEntry) Leave() {
	e.r.leave <- e.id
}
