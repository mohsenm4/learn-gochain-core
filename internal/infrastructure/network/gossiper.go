package network

type Gossiper interface {
	Gossip(msg Message)
}
