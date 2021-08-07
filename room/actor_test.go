package room

import (
	"context"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestActor_Leave(t *testing.T) {
	r := NewRoom()
	defer r.Stop()

	a := NewActor()
	defer a.Stop()

	a.JoinTo(r)
	require.True(t, a.InRoom())

	ok := a.Leave()
	assert.True(t, ok)

	assert.False(t, a.InRoom())

	_, ok = <-a.Inbox()
	assert.False(t, ok)

	ok = a.Leave()
	assert.False(t, ok)
}

func TestActor_BroadcastToRoom(t *testing.T) {
	r := NewRoom()
	defer r.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	a1 := NewActor()
	defer a1.Stop()

	a2 := NewActor()
	defer a2.Stop()

	a3 := NewActor()
	defer a3.Stop()

	a1.JoinTo(r)
	a2.JoinTo(r)
	a3.JoinTo(r)

	body := make([]byte, 1024)
	rand.Read(body)
	a3.BroadcastToRoom(Payload{0x01, body})

	n := 0
L:
	for {
		select {
		case <-ctx.Done():
			t.Fatal(ctx.Err())
			return
		case m := <-a1.Inbox():
			assert.Equal(t, a3.ActorID(), m.Sender)
			assert.EqualValues(t, 0x01, m.Code)
			assert.Equal(t, body, m.Payload)
			n += 1
		case m := <-a2.Inbox():
			assert.Equal(t, a3.ActorID(), m.Sender)
			assert.EqualValues(t, 0x01, m.Code)
			assert.Equal(t, body, m.Payload)
			n += 1
		case m := <-a3.Inbox():
			assert.Equal(t, a3.ActorID(), m.Sender)
			assert.EqualValues(t, 0x01, m.Code)
			assert.Equal(t, body, m.Payload)
			n += 1
		default:
			if n == 3 {
				break L
			}
		}
	}

	a2.Leave()

	body = make([]byte, 1024)
	rand.Read(body)
	a3.BroadcastToRoom(Payload{0x02, body})

	n = 0
M:
	for {
		select {
		case <-ctx.Done():
			t.Fatal(ctx.Err())
			return
		case m := <-a1.Inbox():
			assert.Equal(t, a3.ActorID(), m.Sender)
			assert.EqualValues(t, 0x02, m.Code)
			assert.Equal(t, body, m.Payload)
			n += 1
		case m := <-a3.Inbox():
			assert.Equal(t, a3.ActorID(), m.Sender)
			assert.EqualValues(t, 0x02, m.Code)
			assert.Equal(t, body, m.Payload)
			n += 1
		default:
			if n >= 2 {
				break M
			}
		}
	}

	a4 := NewActor()
	defer a4.Stop()

	a4.JoinTo(r)

	body = make([]byte, 1024)
	rand.Read(body)
	a4.BroadcastToRoom(Payload{0x03, body})

	n = 0
N:
	for {
		select {
		case <-ctx.Done():
			t.Fatal(ctx.Err())
			return
		case m := <-a1.Inbox():
			assert.Equal(t, a4.ActorID(), m.Sender)
			assert.EqualValues(t, 0x03, m.Code)
			assert.Equal(t, body, m.Payload)
			n += 1
		case m := <-a3.Inbox():
			assert.Equal(t, a4.ActorID(), m.Sender)
			assert.EqualValues(t, 0x03, m.Code)
			assert.Equal(t, body, m.Payload)
			n += 1
		case m := <-a4.Inbox():
			assert.Equal(t, a4.ActorID(), m.Sender)
			assert.EqualValues(t, 0x03, m.Code)
			assert.Equal(t, body, m.Payload)
			n += 1
		default:
			if n == 3 {
				break N
			}
		}
	}
}
