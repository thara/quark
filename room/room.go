package room

import "quark"

type Message struct {
	Sender  quark.ActorID
	Code    uint32
	Payload []byte
}

type Subscription chan<- Message

type Room struct {
	join     chan<- Subscription
	leave    chan<- Subscription
	messages chan<- Message

	done chan<- interface{}
}

func NewRoom() *Room {
	messages := make(chan Message, 16)

	join := make(chan Subscription)
	leave := make(chan Subscription)

	done := make(chan interface{})

	go func() {
		defer close(join)
		defer close(leave)
		defer close(messages)
		defer close(done)

		subscribers := map[Subscription]bool{}

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

func (r *Room) Join(s Subscription) {
	r.join <- s
}

func (r *Room) Leave(s Subscription) {
	r.leave <- s
}

func (r *Room) Send(m Message) {
	r.messages <- m
}

func (r *Room) Stop() {
	r.done <- struct{}{}
}
