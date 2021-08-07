package server

import (
	"context"
	"log"
	"math/rand"
	"net"
	"quark/proto"
	"quark/room"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

func TestRoomServer_CreateRoom(t *testing.T) {
	roomServer := &roomServer{
		roomList: room.NewRoomList(),
	}

	ctx := context.Background()
	lis := listenServer(ctx, roomServer)
	conn, err := grpc.DialContext(ctx, "bufnet", listenDialOption(lis), grpc.WithInsecure())
	require.NoError(t, err)
	cli := proto.NewRoomClient(conn)

	roomName := "xxxxxxxx"

	resp, err := cli.CreateRoom(ctx, &proto.CreateRoomRequest{
		RoomName: roomName,
	})
	require.NoError(t, err)

	require.NotNil(t, resp)
	assert.Positive(t, resp.RoomID)
	assert.False(t, resp.AlreadyExist)
	assert.Len(t, roomServer.roomList.Rooms(), 1)

	roomID := resp.RoomID

	resp, err = cli.CreateRoom(ctx, &proto.CreateRoomRequest{
		RoomName: roomName,
	})
	require.NoError(t, err)

	require.NotNil(t, resp)
	assert.Equal(t, roomID, resp.RoomID)
	assert.True(t, resp.AlreadyExist)
	assert.Len(t, roomServer.roomList.Rooms(), 1)
}

func TestRoomServer_Service(t *testing.T) {
	roomServer := &roomServer{
		roomList: room.NewRoomList(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	lis := bufconn.Listen(1024 * 1024)
	s := grpc.NewServer()
	proto.RegisterRoomServer(s, roomServer)
	go func() {
		if err := s.Serve(lis); err != nil {
			log.Fatalf("Server exited with error: %v", err)
		}
	}()
	conn1, err := grpc.DialContext(ctx, "b1", listenDialOption(lis), grpc.WithInsecure())
	require.NoError(t, err)
	conn2, err := grpc.DialContext(ctx, "b2", listenDialOption(lis), grpc.WithInsecure())
	require.NoError(t, err)
	conn3, err := grpc.DialContext(ctx, "b2", listenDialOption(lis), grpc.WithInsecure())
	require.NoError(t, err)

	c1 := proto.NewRoomClient(conn1)
	c2 := proto.NewRoomClient(conn2)
	c3 := proto.NewRoomClient(conn3)

	resp, err := c1.CreateRoom(ctx, &proto.CreateRoomRequest{
		RoomName: "xxxxxx",
	})
	require.NoError(t, err)

	roomID := resp.RoomID

	s1, err := c1.Service(ctx)
	require.NoError(t, err)
	s2, err := c2.Service(ctx)
	require.NoError(t, err)
	s3, err := c3.Service(ctx)
	require.NoError(t, err)

	// c1: join
	var senderActorID string
	{
		err := s1.Send(&proto.Command{
			CommandType: &proto.Command_JoinRoom{
				JoinRoom: &proto.JoinRoom{
					RoomID: roomID,
				},
			},
		})
		require.NoError(t, err)

		m, err := s1.Recv()
		require.NoError(t, err)

		assert.IsType(t, m.EventType, &proto.Event_JoinRoomSucceed{})
		senderActorID = m.EventType.(*proto.Event_JoinRoomSucceed).JoinRoomSucceed.ActorID
	}
	// c2: join
	{
		err := s2.Send(&proto.Command{
			CommandType: &proto.Command_JoinRoom{
				JoinRoom: &proto.JoinRoom{
					RoomID: roomID,
				},
			},
		})
		require.NoError(t, err)

		m, err := s2.Recv()
		require.NoError(t, err)

		assert.IsType(t, m.EventType, &proto.Event_JoinRoomSucceed{})
	}
	// c3: join
	{
		err := s3.Send(&proto.Command{
			CommandType: &proto.Command_JoinRoom{
				JoinRoom: &proto.JoinRoom{
					RoomID: roomID,
				},
			},
		})
		require.NoError(t, err)

		m, err := s3.Recv()
		require.NoError(t, err)

		assert.IsType(t, m.EventType, &proto.Event_JoinRoomSucceed{})
	}

	// c1: send msg
	sendMsg := func() (code uint32, payload []byte) {
		code = rand.Uint32()
		payload = make([]byte, 100)
		rand.Read(payload)

		err := s1.Send(&proto.Command{
			CommandType: &proto.Command_SendMessage{
				SendMessage: &proto.SendMessage{
					Message: &proto.Message{
						Code:    code,
						Payload: payload,
					},
				},
			},
		})
		require.NoError(t, err)
		return
	}
	code, payload := sendMsg()

	// c2: recv msg
	{
		m, err := s2.Recv()
		require.NoError(t, err)
		assert.IsType(t, m.EventType, &proto.Event_MessageReceived{})

		ev := m.EventType.(*proto.Event_MessageReceived)
		msg := ev.MessageReceived

		assert.Equal(t, senderActorID, msg.SenderID)
		assert.Equal(t, code, msg.Message.Code)
		assert.Equal(t, payload, msg.Message.Payload)
	}
	// c3: recv msg
	{
		m, err := s3.Recv()
		require.NoError(t, err)
		assert.IsType(t, m.EventType, &proto.Event_MessageReceived{})

		ev := m.EventType.(*proto.Event_MessageReceived)
		msg := ev.MessageReceived

		assert.Equal(t, senderActorID, msg.SenderID)
		assert.Equal(t, code, msg.Message.Code)
		assert.Equal(t, payload, msg.Message.Payload)
	}

	// c3: leave
	{
		err := s3.Send(&proto.Command{
			CommandType: &proto.Command_LeaveRoom{
				LeaveRoom: &proto.LeaveRoom{},
			},
		})
		require.NoError(t, err)

		m, err := s3.Recv()
		require.NoError(t, err)
		assert.IsType(t, m.EventType, &proto.Event_LeaveRoomSucceed{})
	}

	// c1: send msg 2
	code, payload = sendMsg()

	// c3: never recv
	go func() {
		_, _ = s3.Recv() // blocking
		t.Fail()
	}()

	<-ctx.Done()
}

func listenServer(ctx context.Context, rs proto.RoomServer) *bufconn.Listener {
	lis := bufconn.Listen(1024 * 1024)
	s := grpc.NewServer()
	proto.RegisterRoomServer(s, rs)
	go func() {
		if err := s.Serve(lis); err != nil {
			log.Fatalf("Server exited with error: %v", err)
		}
	}()
	return lis
}

func listenDialOption(lis *bufconn.Listener) grpc.DialOption {
	return grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
		return lis.Dial()
	})
}
