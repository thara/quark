package quark

import "github.com/google/uuid"

type ActorID string

func NewActorID() ActorID {
	return ActorID(uuid.Must(uuid.NewRandom()).String())
}

func (a ActorID) String() string {
	return string(a)
}

type Actor struct {
	id ActorID

	setEntry chan<- *RoomEntry
	getEntry chan<- getActorEntry

	quit chan<- interface{}
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

	setEntry := make(chan *RoomEntry)
	getEntry := make(chan getActorEntry)
	quit := make(chan interface{})

	go func() {
		defer close(setEntry)
		defer close(getEntry)

		var entry *RoomEntry
		for {
			select {
			case <-quit:
				return
			case e := <-setEntry:
				entry = e
			case g := <-getEntry:
				g.out <- entry
			}
		}
	}()

	return &Actor{
		id:       actorID,
		setEntry: setEntry,
		getEntry: getEntry,
		quit:     quit,
	}
}

func (a *Actor) ActorID() ActorID {
	return a.id
}

func (a *Actor) JoinTo(r *Room) {
	e := r.NewEntry()
	a.setEntry <- e
}

func (a *Actor) roomEntry() *RoomEntry {
	out := make(chan *RoomEntry)
	defer close(out)
	a.getEntry <- getActorEntry{out: out}
	return <-out
}

func (a *Actor) Leave() bool {
	e := a.roomEntry()
	if e == nil {
		return false
	}
	e.Leave()
	a.setEntry <- nil
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

func (a *Actor) Stop() {
	close(a.quit)
}

func (a *Actor) InRoom() bool {
	return a.roomEntry() != nil
}

func (a *Actor) IsOwnMessage(m *Message) bool {
	return m.Sender == a.ActorID()
}
