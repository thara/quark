package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"quark/gameserver"

	"go.uber.org/atomic"
	"google.golang.org/grpc"
)

var serverAddr string
var name string
var roomName string

func init() {
	flag.StringVar(&serverAddr, "addr", "127.0.0.1:20000", "server address")
	flag.StringVar(&name, "name", "Tom", "Your name")
	flag.StringVar(&roomName, "roomName", "sample", "room name")
}

func main() {
	flag.Parse()

	var opts []grpc.DialOption
	opts = append(opts, grpc.WithInsecure())

	conn, err := grpc.Dial(serverAddr, opts...)
	if err != nil {
		log.Fatalf("fail to dial: %v", err)
	}
	defer conn.Close()

	cli := gameserver.NewRoomClient(conn)

	ctx := context.Background()

	resp, err := cli.CreateRoom(ctx, &gameserver.CreateRoomRequest{
		RoomName: roomName,
	})
	if err != nil {
		log.Fatalf("fail to create room: %v", err)
	}
	if resp.AlreadyExist {
		fmt.Printf("room: %s is already exist\n", roomName)
	}

	roomID := resp.RoomID

	stream, err := cli.Service(ctx)
	if err != nil {
		log.Fatalf("fail to service: %v", err)
	}

	if err := stream.Send(&gameserver.Command{
		CommandType: &gameserver.Command_JoinRoom{
			JoinRoom: &gameserver.JoinRoom{
				RoomID: roomID,
			},
		},
	}); err != nil {
		log.Fatalf("Failed to leave room: %v", err)
	}

	joined := make(chan interface{})
	defer close(joined)

	var actorID atomic.String

	go func() {
		// var actorID string
		for {
			in, err := stream.Recv()
			if err == io.EOF {
				return
			} else if err != nil {
				log.Fatalf("Failed to receive a note : %v", err)
			}

			switch ev := in.EventType.(type) {
			case *gameserver.Event_JoinRoomSucceed:
				id := ev.JoinRoomSucceed.ActorID
				fmt.Printf("You are %s\n", id)

				fmt.Printf("%s > ", id)
				actorID.Store(id)

				joined <- struct{}{}
			case *gameserver.Event_LeaveRoomSucceed:
				fmt.Println("bye.")
				os.Exit(0)
			case *gameserver.Event_MessageReceived:
				recvMsg := ev.MessageReceived
				cmd := parseCmd(recvMsg.SenderID, recvMsg.Message.Code, recvMsg.Message.Payload)
				cmd.display()

				switch cmd.(type) {
				case *ping:
					msg := pongCmd()
					if err := sendMessage(stream, msg); err != nil {
						log.Fatalf("Failed to leave room: %v", err)
					}
					p := pong{actorID: actorID.Load()}
					p.display()
				}

				fmt.Printf("%s > ", actorID.Load())
			}
		}
	}()

	<-joined

	stdin := bufio.NewScanner(os.Stdin)
	for stdin.Scan() {
		text := stdin.Text()

		switch text {
		case "!exit":
			if err := stream.Send(&gameserver.Command{
				CommandType: &gameserver.Command_LeaveRoom{
					LeaveRoom: &gameserver.LeaveRoom{},
				},
			}); err != nil {
				log.Fatalf("Failed to leave room: %v", err)
			}
		case "!ping":
			msg := pingCmd()
			if err := sendMessage(stream, msg); err != nil {
				log.Fatalf("Failed to leave room: %v", err)
			}
			fmt.Printf("%s > ", actorID.Load())
		default:
			msg := textCmd(text)
			if err := sendMessage(stream, msg); err != nil {
				log.Fatalf("Failed to leave room: %v", err)
			}
			fmt.Printf("%s > ", actorID.Load())
		}
	}
}

type cmdType = uint32

var (
	CmdPing cmdType = 0x10
	CmdPong cmdType = 0x11
	CmdText cmdType = 0x20
)

type cmd interface {
	display()
}

type ping struct{ actorID string }

func (p *ping) display() { fmt.Printf("\033[0G%s > ping!\n", p.actorID) }

type pong struct{ actorID string }

func (p *pong) display() { fmt.Printf("\033[0G%s > pong!\n", p.actorID) }

type text struct {
	senderID string
	text     string
}

func (t *text) display() {
	fmt.Printf("\033[0G%s > %s\n", t.senderID, t.text)
}

func pingCmd() *gameserver.Message {
	return &gameserver.Message{
		Code: CmdPing,
	}
}

func pongCmd() *gameserver.Message {
	return &gameserver.Message{
		Code: CmdPong,
	}
}

func textCmd(text string) *gameserver.Message {
	return &gameserver.Message{
		Code:    CmdText,
		Payload: []byte(text),
	}
}

func parseCmd(senderID string, code uint32, payload []byte) cmd {
	switch code {
	case CmdPing:
		return &ping{actorID: senderID}
	case CmdPong:
		return &pong{actorID: senderID}
	case CmdText:
		return &text{senderID: senderID, text: string(payload)}
	default:
		return nil
	}
}

func sendMessage(stream gameserver.Room_ServiceClient, m *gameserver.Message) error {
	return stream.Send(&gameserver.Command{
		CommandType: &gameserver.Command_SendMessage{
			SendMessage: &gameserver.SendMessage{
				Message: m,
			},
		},
	})
}
