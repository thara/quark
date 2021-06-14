package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/pkg/errors"
)

const maxBufferSize = 1024

const writeTimeout = 3 * time.Second
const peerTimeout = 10 * time.Second

func run(ctx context.Context, addr string) error {
	pc, err := net.ListenPacket("udp", addr)
	if err != nil {
		return errors.WithStack(err)
	}
	defer pc.Close()

	fmt.Printf("Listening at %s\n", addr)

	doneCh := make(chan error, 1)
	buf := make([]byte, maxBufferSize)

	rm := NewRoom()

	go func() {
		for {
			n, srcAddr, err := pc.ReadFrom(buf)
			if err != nil {
				doneCh <- errors.WithStack(err)
				return
			}
			fmt.Printf("received: bytes=%d from=%s\n", n, srcAddr.String())

			id := PeerID(srcAddr.String())

			src, ok := rm.GetPeer(id)
			if !ok {
				src = rm.AddPeerTo(srcAddr, id)
				fmt.Printf("Add peer: %s\n", srcAddr.String())
			}

			now := time.Now()
			src.expiry = now.Add(peerTimeout)

			deadline := now.Add(writeTimeout)
			if err := pc.SetWriteDeadline(deadline); err != nil {
				doneCh <- errors.WithStack(err)
				return
			}

			for _, p := range rm.Peers() {
				if p.id == id {
					continue
				}

				n, err := p.Write(pc, buf[:n])
				if err != nil {
					continue
				}
				fmt.Printf("write: bytes=%d from=%s\n", n, p.addr.String())
			}
		}
	}()

	// peer timeout
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	go func() {
		for {
			select {
			case t := <-ticker.C:
				for _, p := range rm.Peers() {
					if t.After(p.expiry) {
						if rm.RemovePeer(p.id) {
							p.Write(pc, []byte("timeout."))
						}
					}
				}
			}
		}
	}()

	select {
	case <-ctx.Done():
		fmt.Println("cancelled")
		return ctx.Err()
	case err := <-doneCh:
		return err
	}
}

func main() {
	ctx := context.Background()

	if err := run(ctx, "127.0.0.1:28080"); err != nil {
		log.Fatal(err)
	}
}
