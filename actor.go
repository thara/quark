package quark

import (
	"sync"

	"github.com/google/uuid"
)

type ActorID string

func NewActorID() ActorID {
	return ActorID(uuid.Must(uuid.NewRandom()).String())
}

func (a ActorID) String() string {
	return string(a)
}

type Actor struct {
	id ActorID

	re *RoomEntry
	mu sync.RWMutex
}

type Payload struct {
	Code uint32
	Body []byte
}

type getActorEntry struct {
	out chan<- *RoomEntry
}

func NewActor() *Actor {
	actorID := NewActorID()
	return &Actor{id: actorID}
}

func (a *Actor) ActorID() ActorID {
	return a.id
}

func (a *Actor) JoinTo(r *Room) {
	e := r.NewEntry()

	a.mu.Lock()
	defer a.mu.Unlock()
	a.re = e
}

func (a *Actor) roomEntry() *RoomEntry {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.re
}

func (a *Actor) Leave() bool {
	e := a.roomEntry()
	if e == nil {
		return false
	}

	a.mu.Lock()
	defer a.mu.Unlock()
	e.Leave()
	a.re = nil
	return true
}

func (a *Actor) BroadcastToRoom(p Payload) bool {
	e := a.roomEntry()
	if e == nil {
		return false
	}
	e.Send(Message{
		Sender:  a.id,
		Code:    p.Code,
		Payload: p.Body,
	})
	return true
}

func (a *Actor) Inbox() <-chan Message {
	e := a.roomEntry()
	if e == nil {
		c := make(chan Message)
		close(c)
		return c
	}
	return a.roomEntry().Subscription()
}

func (a *Actor) InRoom() bool {
	return a.roomEntry() != nil
}

func (a *Actor) IsOwnMessage(m *Message) bool {
	return m.Sender == a.ActorID()
}
