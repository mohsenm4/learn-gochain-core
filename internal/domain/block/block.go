package block

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"time"

	"github.com/Mohsen20031203/learn-gochain-core/internal/domain/transaction"
)

type Block struct {
	Index        int                       `json:"index"`
	Timestamp    time.Time                 `json:"timestamp"`
	Transactions []transaction.Transaction `json:"transactions"`
	Hash         string                    `json:"hash"`
	PrevHash     string                    `json:"prev_hash"`
	Nonce        int                       `json:"nonce"`
}

func NewBlock(index int, transactions []transaction.Transaction, prevHash string) *Block {
	block := &Block{
		Index:        index,
		Timestamp:    time.Now(),
		Transactions: transactions,
		PrevHash:     prevHash,
		Nonce:        0,
	}
	return block
}

func (b *Block) CalculateHash() string {
	data, _ := json.Marshal(b.Transactions)
	record :=
		strconv.FormatInt(int64(b.Index), 10) +
			strconv.FormatInt(b.Timestamp.UnixNano(), 10) +
			string(data) +
			b.PrevHash +
			strconv.FormatInt(int64(b.Nonce), 10)

	hash := sha256.Sum256([]byte(record))
	return hex.EncodeToString(hash[:])
}

func (b *Block) IsValid(prev Block) bool {
	if b.PrevHash != prev.Hash {
		return false
	}
	if b.CalculateHash() != b.Hash {
		return false
	}
	return true
}
