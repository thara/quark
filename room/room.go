package room

import "quark"

type Message struct {
	Sender  quark.ActorID
	Code    uint32
	Payload []byte
}

type Subscriber chan<- Message

type Room struct {
	join     chan<- Subscriber
	leave    chan<- Subscriber
	messages chan<- Message

	done chan<- interface{}
}

func NewRoom() *Room {
	messages := make(chan Message, 16)

	join := make(chan Subscriber)
	leave := make(chan Subscriber)

	done := make(chan interface{})

	go func() {
		defer close(join)
		defer close(leave)
		defer close(messages)
		defer close(done)

		subscribers := map[Subscriber]bool{}

		for {
			select {
			case <-done:
				return
			case s := <-join:
				subscribers[s] = true
			case s := <-leave:
				delete(subscribers, s)
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

func (r *Room) Join(s Subscriber) {
	r.join <- s
}

func (r *Room) Leave(s Subscriber) {
	r.leave <- s
}

func (r *Room) Send(m Message) {
	r.messages <- m
}

func (r *Room) Stop() {
	r.done <- struct{}{}
}
