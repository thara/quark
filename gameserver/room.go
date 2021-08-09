package gameserver

type Message interface{}

type ActorMessage struct {
	Sender  ActorID
	Code    uint32
	Payload []byte
}

type subscription chan<- Message

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

type Room struct {
	join     chan<- roomJoinCmd
	leave    chan<- ActorID
	messages chan<- ActorMessage

	done chan<- interface{}
}

type roomJoinCmd struct {
	actorID ActorID
	out     chan<- (chan Message)
}

func NewRoom() *Room {
	messages := make(chan ActorMessage, 16)

	join := make(chan roomJoinCmd)
	leave := make(chan ActorID)

	done := make(chan interface{})

	go func() {
		defer close(join)
		defer close(leave)
		defer close(messages)
		defer close(done)

		subscribers := map[ActorID]subscription{}

		currentActors := func() []ActorID {
			s := make([]ActorID, 0, len(subscribers))
			for id := range subscribers {
				s = append(s, id)
			}
			return s
		}

		for {
			select {
			case <-done:
				return
			case cmd := <-join:
				s := make(chan Message, 128)
				subscribers[cmd.actorID] = s
				cmd.out <- s

				ev := JoinRoomEvent{
					ActorList: currentActors(),
					NewActor:  cmd.actorID,
				}
				for id, other := range subscribers {
					if id != cmd.actorID {
						other <- ev
					}
				}
			case id := <-leave:
				if s, ok := subscribers[id]; ok {
					delete(subscribers, id)
					close(s)

					ev := LeaveRoomEvent{
						ActorList:    currentActors(),
						RemovedActor: id,
					}
					for _, other := range subscribers {
						other <- ev
					}
				}
			case m := <-messages:
				for _, s := range subscribers {
					s <- m
				}
			}
		}
	}()
	return &Room{
		join: join, leave: leave, messages: messages,
		done: done,
	}
}

func (r *Room) NewEntry(actorID ActorID) *RoomEntry {
	out := make(chan (chan Message))
	defer close(out)
	r.join <- roomJoinCmd{actorID: actorID, out: out}
	return &RoomEntry{id: actorID, r: r, s: <-out}
}

func (r *Room) Stop() {
	r.done <- struct{}{}
}
