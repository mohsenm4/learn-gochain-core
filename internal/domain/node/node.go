package node

import (
	"sync"

	"github.com/Mohsen20031203/learn-gochain-core/internal/domain/blockchain"
	"github.com/Mohsen20031203/learn-gochain-core/internal/domain/mempool"
	"github.com/Mohsen20031203/learn-gochain-core/internal/domain/utxo"
)

type Node struct {
	id      string
	chain   *blockchain.Blockchain
	mempool *mempool.Mempool
	utxo    *utxo.Set

	txIndexMu sync.RWMutex
	txIndex   map[string]int // txID -> block index containing it
}

// NewNode creates and returns a new Node instance.
// id is the unique identifier for the node ,
// difficulty is the mining difficulty for the node's blockchain
func NewNode(id string, difficulty int) *Node {
	return &Node{
		chain:   blockchain.New(difficulty),
		id:      id,
		mempool: mempool.NewMempool(),
		utxo:    utxo.NewSet(),
		txIndex: make(map[string]int),
	}
}

// GetID returns the ID of the node
func (n *Node) GetID() string {
	return n.id
}

// IndexTx records that a transaction was confirmed in the given block index.
func (n *Node) IndexTx(txID string, blockIndex int) {
	n.txIndexMu.Lock()
	defer n.txIndexMu.Unlock()
	n.txIndex[txID] = blockIndex
}

// TxBlockIndex returns the block index a transaction was confirmed in.
func (n *Node) TxBlockIndex(txID string) (int, bool) {
	n.txIndexMu.RLock()
	defer n.txIndexMu.RUnlock()
	idx, ok := n.txIndex[txID]
	return idx, ok
}

// Confirmations returns the number of blocks built on top of and including
// the block containing txID. 0 means unknown/unconfirmed.
func (n *Node) Confirmations(txID string) int {
	idx, ok := n.TxBlockIndex(txID)
	if !ok {
		return 0
	}
	return n.CountBlocksinChain() - idx
}
