package quark

import "github.com/google/uuid"

type ActorID string

func NewActorID() ActorID {
	return ActorID(uuid.Must(uuid.NewRandom()).String())
}

func (a ActorID) String() string {
	return string(a)
}
