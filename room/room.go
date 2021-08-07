package room

import (
	"quark"
)

type Message struct {
	Sender  quark.ActorID
	Code    uint32
	Payload []byte
}

type subscription chan<- Message

type RoomEntry struct {
	r *Room
	s chan Message
}

func (e *RoomEntry) Subscription() <-chan Message {
	return e.s
}

func (e *RoomEntry) Send(m Message) {
	e.r.messages <- m
}

func (e *RoomEntry) Leave() {
	e.r.leave <- e.s
}

type Room struct {
	join     chan<- roomJoinCmd
	leave    chan<- subscription
	messages chan<- Message

	done chan<- interface{}
}

type roomJoinCmd struct {
	out chan<- (chan Message)
}

func NewRoom() *Room {
	messages := make(chan Message, 16)

	join := make(chan roomJoinCmd)
	leave := make(chan subscription)

	done := make(chan interface{})

	go func() {
		defer close(join)
		defer close(leave)
		defer close(messages)
		defer close(done)

		subscribers := map[subscription]bool{}

		for {
			select {
			case <-done:
				return
			case cmd := <-join:
				s := make(chan Message)
				subscribers[s] = true
				cmd.out <- s
			case s := <-leave:
				delete(subscribers, s)
				close(s)
			case m := <-messages:
				for s := range subscribers {
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

func (r *Room) NewEntry() *RoomEntry {
	out := make(chan (chan Message))
	defer close(out)
	r.join <- roomJoinCmd{out: out}
	return &RoomEntry{r: r, s: <-out}
}

func (r *Room) Leave(e *RoomEntry) {
	r.leave <- e.s
}

func (r *Room) Send(m Message) {
	r.messages <- m
}

func (r *Room) Stop() {
	r.done <- struct{}{}
}
