package blockchain

import (
	"fmt"

	"github.com/Mohsen20031203/learn-gochain-core/internal/domain/block"
)

// Without this, a peer could mint unlimited coins by claiming a higher subsidy.
func (s *NodeService) validateCoinbaseReward(blc block.Block) bool {
	coinbase := blc.Transactions[0]
	var claimed uint64
	for _, out := range coinbase.Outputs {
		claimed += out.Value
	}

	var fees uint64
	for i := 1; i < len(blc.Transactions); i++ {
		fees += s.node.Fee(&blc.Transactions[i])
	}

	allowed := blockReward(blc.Index) + fees
	if claimed > allowed {
		fmt.Printf("Invalid block: coinbase claims %d, allowed %d (subsidy=%d fees=%d)\n",
			claimed, allowed, blockReward(blc.Index), fees)
		return false
	}
	return true
}

func (s *NodeService) validataBlock(blc block.Block) bool {
	if len(blc.Transactions) == 0 {
		fmt.Println("Invalid block: no transactions (coinbase required)")
		return false
	}
	if !blc.Transactions[0].IsCoinbase() {
		fmt.Println("Invalid block: first transaction must be coinbase")
		return false
	}

	lastBlockHash := s.node.GetChainLastBlockHash()
	if lastBlockHash == "" {
		if blc.Index != 0 {
			fmt.Println("Invalid block: genesis block index must be 0")
			return false
		}
		return true
	}

	if !s.node.IsValidNewBlockChain(blc) {
		fmt.Println("Invalid block: previous hash does not match")
		return false
	}
	if blc.Index < s.node.CountBlocksinChain() {
		fmt.Println("Invalid block: index is not greater than last block index")
		return false
	}
	if !s.node.IsValidPoW(&blc) {
		fmt.Println("Invalid block: proof of work is not valid")
		return false
	}
	if !s.validateCoinbaseReward(blc) {
		return false
	}

	b, err := s.repo.Get(blc.Hash)
	if err == nil && b.Hash != "" {
		fmt.Println("Invalid block: block already exists")
		return false
	}
	return true
}
