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

func NewNode(id string, difficulty int) *Node {
	return &Node{
		chain:   blockchain.New(difficulty),
		id:      id,
		mempool: mempool.NewMempool(),
		utxo:    utxo.NewSet(),
		txIndex: make(map[string]int),
	}
}

func (n *Node) GetID() string {
	return n.id
}

func (n *Node) IndexTx(txID string, blockIndex int) {
	n.txIndexMu.Lock()
	defer n.txIndexMu.Unlock()
	n.txIndex[txID] = blockIndex
}

func (n *Node) TxBlockIndex(txID string) (int, bool) {
	n.txIndexMu.RLock()
	defer n.txIndexMu.RUnlock()
	idx, ok := n.txIndex[txID]
	return idx, ok
}

// Blocks built on top of and including the one holding txID; 0 if unknown.
func (n *Node) Confirmations(txID string) int {
	idx, ok := n.TxBlockIndex(txID)
	if !ok {
		return 0
	}
	return n.CountBlocksinChain() - idx
}
