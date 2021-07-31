package room

import "quark"

type message struct {
	sender  quark.ActorID
	code    uint32
	payload []byte
}

type subscriber chan<- message

type room struct {
	join     chan<- subscriber
	leave    chan<- subscriber
	messages chan<- message

	done chan<- interface{}
}

func newRoom() *room {
	messages := make(chan message, 16)

	join := make(chan subscriber)
	leave := make(chan subscriber)

	done := make(chan interface{})

	go func() {
		defer close(join)
		defer close(leave)
		defer close(messages)
		defer close(done)

		subscribers := map[subscriber]bool{}

		for {
			select {
			case <-done:
				return
			case s := <-join:
				subscribers[s] = true
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
	return &room{
		join: join, leave: leave, messages: messages,
		done: done,
	}
}

func (r *room) Join(s subscriber) {
	r.join <- s
}

func (r *room) Leave(s subscriber) {
	r.leave <- s
}

func (r *room) Send(m message) {
	r.messages <- m
}

func (r *room) Stop() {
	r.done <- struct{}{}
}
