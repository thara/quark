package main

import (
	"net"
	"sync"
	"time"
)

type PeerID string

func (i PeerID) String() string {
	return string(i)
}

type Peer struct {
	id   PeerID
	addr net.Addr

	expiry time.Time
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

func (r *Room) GetPeer(id PeerID) (*Peer, bool) {
	r.mux.RLock()
	defer r.mux.RUnlock()
	idx, ok := r.peerIdx[id.String()]
	if !ok {
		return nil, false
	}
	return r.peers[idx], true
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

func (r *Room) RemovePeer(id PeerID) bool {
	key := id.String()

	r.mux.Lock()
	defer r.mux.Unlock()

	idx, ok := r.peerIdx[key]
	if !ok {
		return false
	}
	delete(r.peerIdx, key)

	last := r.peers[len(r.peers)-1]
	if last.id != id {
		r.peers[idx] = last
		r.peerIdx[last.id.String()] = idx
	}
	r.peers = r.peers[:len(r.peers)-1]

	return true
}

func (r *Room) Peers() []*Peer {
	return r.peers[:]
}
