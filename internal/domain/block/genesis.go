package block

import (
	"time"

	"github.com/Mohsen20031203/learn-gochain-core/internal/domain/transaction"
)

// Fixed so every node produces the same genesis hash.
const (
	GenesisAddress = "genesis"
	GenesisReward  = uint64(50)
)

var GenesisTimestamp = time.Unix(1735689600, 0).UTC()

func NewGenesis() *Block {
	coinbase := transaction.NewCoinbase(GenesisAddress, GenesisReward, GenesisTimestamp.UnixNano())

	b := &Block{
		Index:        0,
		Timestamp:    GenesisTimestamp,
		Transactions: []transaction.Transaction{*coinbase},
		PrevHash:     "0",
		Nonce:        0,
	}
	b.Hash = b.CalculateHash()
	return b
}
