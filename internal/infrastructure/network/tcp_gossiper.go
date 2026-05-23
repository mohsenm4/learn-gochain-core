package network

import (
	"encoding/json"
	"net"
	"sync"
)

type TCPGossiper struct {
	mu      sync.RWMutex
	peers   map[string]struct{}
	ownAddr string
}

func NewTCPGossiper(peers []string, ownAddr string) *TCPGossiper {
	m := make(map[string]struct{}, len(peers))
	for _, p := range peers {
		if p != "" && p != ownAddr {
			m[p] = struct{}{}
		}
	}
	return &TCPGossiper{peers: m, ownAddr: ownAddr}
}

func (g *TCPGossiper) Gossip(msg Message) {
	for _, peer := range g.Peers() {
		conn, err := net.Dial("tcp", peer)
		if err != nil {
			continue
		}
		json.NewEncoder(conn).Encode(msg)
		conn.Close()
	}
}

// Returns true if the peer was newly added.
func (g *TCPGossiper) AddPeer(addr string) bool {
	if addr == "" || addr == g.ownAddr {
		return false
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if _, exists := g.peers[addr]; exists {
		return false
	}
	g.peers[addr] = struct{}{}
	return true
}

func (g *TCPGossiper) Peers() []string {
	g.mu.RLock()
	defer g.mu.RUnlock()
	out := make([]string, 0, len(g.peers))
	for p := range g.peers {
		out = append(out, p)
	}
	return out
}
