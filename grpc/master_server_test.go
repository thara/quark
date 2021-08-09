package grpc

import (
	"context"
	"log"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"

	"quark/masterserver"
	"quark/proto"
	"quark/proto/primitive"
)

func TestMasterServer(t *testing.T) {
	fleet := masterserver.NewFleet()

	ctx := context.Background()

	var ms proto.MasterServerClient
	{
		masterServer := NewMasterServer(fleet)
		lis := listenMasterServer(ctx, masterServer)

		conn, err := grpc.DialContext(ctx, "bufnet", listenDialOption(lis), grpc.WithInsecure())
		require.NoError(t, err)

		ms = proto.NewMasterServerClient(conn)
	}
	var lobby proto.LobbyClient
	{
		lobbyServer := NewLobbyServer(fleet)
		lis := listenLobbyServer(ctx, lobbyServer)

		conn, err := grpc.DialContext(ctx, "bufnet", listenDialOption(lis), grpc.WithInsecure())
		require.NoError(t, err)

		lobby = proto.NewLobbyClient(conn)
	}

	addr := &primitive.GameServer{Address: "0.0.0.0", Port: "14000"}

	gsStream, err := ms.RegisterGameServer(ctx, &proto.RegisterGameServerRequest{
		NewGameServer: addr,
	})
	require.NoError(t, err)

	{
		m, err := gsStream.Recv()
		require.NoError(t, err)

		assert.IsType(t, m.Message, &proto.MasterServerMessage_Registered{})

		_ = m.Message.(*proto.MasterServerMessage_Registered).Registered.GameServerID
	}

	resp, err := lobby.CreateRoom(ctx, &proto.CreateRoomRequest{
		RoomName: "test",
	})
	require.NoError(t, err)

	lobbyRoomID := resp.RoomID

	var gameserverRoomID uint64
	{
		m, err := gsStream.Recv()
		require.NoError(t, err)

		assert.IsType(t, m.Message, &proto.MasterServerMessage_Allocation{})

		gameserverRoomID = m.Message.(*proto.MasterServerMessage_Allocation).Allocation.Room.RoomID
	}

	assert.Equal(t, gameserverRoomID, lobbyRoomID)
}

func listenMasterServer(ctx context.Context, svr proto.MasterServerServer) *bufconn.Listener {
	lis := bufconn.Listen(1024 * 1024)
	s := grpc.NewServer()
	proto.RegisterMasterServerServer(s, svr)
	go func() {
		if err := s.Serve(lis); err != nil {
			log.Fatalf("Server exited with error: %v", err)
		}
	}()
	return lis
}

func listenLobbyServer(ctx context.Context, svr proto.LobbyServer) *bufconn.Listener {
	lis := bufconn.Listen(1024 * 1024)
	s := grpc.NewServer()
	proto.RegisterLobbyServer(s, svr)
	go func() {
		if err := s.Serve(lis); err != nil {
			log.Fatalf("Server exited with error: %v", err)
		}
	}()
	return lis
}
