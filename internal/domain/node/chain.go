package node

import "github.com/Mohsen20031203/learn-gochain-core/internal/domain/block"

// IsValidNewBlockChain checks if a new block is valid according to the node's blockchain
func (n *Node) IsValidNewBlockChain(blockHash block.Block) bool {
	return n.chain.IsValidNewBlock(&blockHash)
}

// UpdateChain updates the node's blockchain with a new block
func (n *Node) UpdateChain(blockHash block.Block) {
	n.chain.UpdateWithNewBlock(&blockHash)
}

// GetChainLastBlockHash returns the last block hash of the node's blockchain
func (n *Node) GetChainLastBlockHash() string {
	return n.chain.LastBlockHash()
}

// CountBlocksinChain returns the current height of the node's blockchain
func (n *Node) CountBlocksinChain() int {
	return n.chain.CountBlocks()
}

// GetChainDifficulty returns the difficulty of the node's blockchain
func (n *Node) GetChainDifficulty() int {
	return n.chain.GetDifficulty()
}

func (n *Node) MineBlock(b *block.Block) {
	n.chain.Mine(b)
}

func (s *Node) IsValidPoW(b *block.Block) bool {
	return s.chain.IsValidPoW(b)
}

func (n *Node) AdjustDifficulty(actualSeconds int64) {
	n.chain.AdjustDifficulty(actualSeconds)
}
