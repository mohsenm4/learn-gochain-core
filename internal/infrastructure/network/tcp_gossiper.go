package network

import (
	"encoding/json"
	"net"
)

type TCPGossiper struct {
	peers []string
}

func NewTCPGossiper(peers []string) *TCPGossiper {
	return &TCPGossiper{peers: peers}
}

func (g *TCPGossiper) Gossip(msg Message) {
	for _, peer := range g.peers {
		conn, err := net.Dial("tcp", peer)
		if err != nil {
			continue
		}
		json.NewEncoder(conn).Encode(msg)
		conn.Close()
	}
}
