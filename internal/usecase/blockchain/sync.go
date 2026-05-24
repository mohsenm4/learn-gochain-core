package blockchain

import (
	"encoding/json"
	"fmt"

	"github.com/Mohsen20031203/learn-gochain-core/internal/domain/block"
	"github.com/Mohsen20031203/learn-gochain-core/internal/infrastructure/network"
)

type GetChainRequest struct {
	From       string `json:"from"`
	HaveHeight int    `json:"have_height"`
}

func (s *NodeService) RequestChain() {
	if s.gossiper == nil {
		return
	}
	req := GetChainRequest{
		From:       s.config.TCPAddress,
		HaveHeight: s.node.CountBlocksinChain(),
	}
	data, err := json.Marshal(req)
	if err != nil {
		return
	}
	msg := network.Message{Type: "get_chain", Data: data}
	s.markSeen(messageID(msg))
	s.gossiper.Gossip(msg)
}

func (s *NodeService) handleGetChain(req GetChainRequest) {
	if s.gossiper == nil || req.From == "" {
		return
	}
	myHeight := s.node.CountBlocksinChain()
	if myHeight <= req.HaveHeight {
		return
	}
	chain, err := s.GetChain()
	if err != nil {
		return
	}
	var missing []block.Block
	for _, b := range chain {
		if b.Index >= req.HaveHeight {
			missing = append(missing, b)
		}
	}
	if len(missing) == 0 {
		return
	}
	data, err := json.Marshal(missing)
	if err != nil {
		return
	}
	s.gossiper.Send(req.From, network.Message{Type: "chain", Data: data})
}

func (s *NodeService) handleChainResponse(blocks []block.Block) {
	for i := range blocks {
		b := blocks[i]
		if b.Index <= s.node.CountBlocksinChain()-1 {
			continue
		}
		if !s.validataBlock(b) {
			fmt.Println("[ibd] rejected block at index", b.Index)
			return
		}
		if err := s.saveBlock(&b); err != nil {
			fmt.Println("[ibd] save error:", err)
			return
		}
		fmt.Println("[ibd] synced block:", b.Index, b.Hash)
	}
}
