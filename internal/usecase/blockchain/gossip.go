package blockchain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/Mohsen20031203/learn-gochain-core/internal/domain/block"
	"github.com/Mohsen20031203/learn-gochain-core/internal/domain/transaction"
	"github.com/Mohsen20031203/learn-gochain-core/internal/infrastructure/network"
)

func (s *NodeService) SetGossiper(g *network.TCPGossiper) {
	s.gossiper = g
}

// Content hash used to dedupe gossiped messages.
func messageID(msg network.Message) string {
	h := sha256.New()
	h.Write([]byte(msg.Type))
	h.Write([]byte{':'})
	h.Write(msg.Data)
	return hex.EncodeToString(h.Sum(nil))
}

const seenMaxSize = 10000

func (s *NodeService) markSeen(id string) bool {
	s.seenMu.Lock()
	defer s.seenMu.Unlock()
	if _, ok := s.seen[id]; ok {
		return false
	}
	if len(s.seen) >= seenMaxSize {
		// Bounded cache: evict half when full. Crude but prevents unbounded growth.
		i := 0
		for k := range s.seen {
			delete(s.seen, k)
			i++
			if i >= seenMaxSize/2 {
				break
			}
		}
	}
	s.seen[id] = struct{}{}
	return true
}

func (s *NodeService) HandleNodeMessage(msg network.Message) {
	if !s.markSeen(messageID(msg)) {
		return
	}

	switch msg.Type {
	case "block":
		var blc block.Block
		if err := json.Unmarshal(msg.Data, &blc); err != nil {
			fmt.Println("error unmarshall block from node message:", err)
			return
		}
		if !s.validateBlock(blc) {
			fmt.Println("received invalid block from peer")
			return
		}
		if err := s.saveBlock(&blc); err != nil {
			fmt.Println("error saving block from node message:", err)
			return
		}
		fmt.Println("block saved from peer:", blc.Hash)
		s.forward(msg)
	case "tx":
		var txs []transaction.Transaction
		if err := json.Unmarshal(msg.Data, &txs); err != nil {
			fmt.Println("error unmarshall txs from node message:", err)
			return
		}
		if err := s.SubmitTransactions(txs); err != nil {
			fmt.Println("error submitting incoming txs:", err)
		}
		s.forward(msg)
	case "peers":
		var peers []string
		if err := json.Unmarshal(msg.Data, &peers); err != nil {
			fmt.Println("error unmarshall peers:", err)
			return
		}
		s.handlePeerList(peers)
		s.forward(msg)
	case "get_chain":
		var req GetChainRequest
		if err := json.Unmarshal(msg.Data, &req); err != nil {
			fmt.Println("error unmarshall get_chain:", err)
			return
		}
		s.handleGetChain(req)
	case "chain":
		var blocks []block.Block
		if err := json.Unmarshal(msg.Data, &blocks); err != nil {
			fmt.Println("error unmarshall chain:", err)
			return
		}
		s.handleChainResponse(blocks)
	}
}

func (s *NodeService) handlePeerList(peers []string) {
	if s.gossiper == nil {
		return
	}
	added := false
	for _, p := range peers {
		if s.gossiper.AddPeer(p) {
			added = true
			fmt.Println("[peers] discovered:", p)
		}
	}
	if added {
		s.AnnouncePeers()
	}
}

func (s *NodeService) AnnouncePeers() {
	if s.gossiper == nil {
		return
	}
	known := s.gossiper.Peers()
	known = append(known, s.config.PublicAddress)
	data, err := json.Marshal(known)
	if err != nil {
		return
	}
	msg := network.Message{Type: "peers", Data: data}
	s.markSeen(messageID(msg))
	s.gossiper.Gossip(msg)
}

func (s *NodeService) forward(msg network.Message) {
	if s.gossiper == nil {
		return
	}
	s.gossiper.Gossip(msg)
}

// gossipTxs relays newly-accepted transactions to peers. markSeen prevents
// double-broadcasting when these same txs arrived via gossip in the first place.
func (s *NodeService) gossipTxs(txs []transaction.Transaction) {
	if s.gossiper == nil || len(txs) == 0 {
		return
	}
	data, err := json.Marshal(txs)
	if err != nil {
		return
	}
	msg := network.Message{Type: "tx", Data: data}
	if !s.markSeen(messageID(msg)) {
		return
	}
	s.gossiper.Gossip(msg)
}

func (s *NodeService) gossipBlock(blc *block.Block) {
	if s.gossiper == nil {
		return
	}

	data, err := json.Marshal(blc)
	if err != nil {
		return
	}

	msg := network.Message{
		Type: "block",
		Data: data,
	}

	s.markSeen(messageID(msg))
	s.gossiper.Gossip(msg)
}
