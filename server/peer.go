package main

import (
	"net"
	"sync"
)

type PeerID string

func (i PeerID) String() string {
	return string(i)
}

type Peer struct {
	id   PeerID
	addr net.Addr
}

func (p *Peer) Write(pc net.PacketConn, msg []byte) (int, error) {
	return pc.WriteTo(msg[:], p.addr)
}

type Room struct {
	peerIdx map[string]int
	peers   []*Peer

	mux sync.RWMutex
}

func NewRoom() *Room {
	return &Room{
		peerIdx: make(map[string]int),
		mux:     sync.RWMutex{},
	}
}

func (r *Room) HasPeer(id PeerID) bool {
	r.mux.RLock()
	_, ok := r.peerIdx[id.String()]
	r.mux.RUnlock()
	return ok
}

func (r *Room) AddPeerTo(addr net.Addr, id PeerID) *Peer {
	p := Peer{id: id, addr: addr}
	r.mux.Lock()
	r.peers = append(r.peers, &p)
	r.peerIdx[id.String()] = len(r.peers) - 1
	r.mux.Unlock()
	return &p
}

func (r *Room) Peers() []*Peer {
	return r.peers[:]
}
