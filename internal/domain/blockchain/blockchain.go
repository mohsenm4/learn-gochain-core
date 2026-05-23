package blockchain

import (
	"strings"

	"github.com/Mohsen20031203/learn-gochain-core/internal/domain/block"
)

// Difficulty retargets every AdjustmentInterval blocks toward TargetBlockTime.
const (
	TargetBlockTimeSeconds = 10
	AdjustmentInterval     = 5
	MinDifficulty          = 1
)

type Blockchain struct {
	difficulty int
	lastHash   string
	height     int
}

func New(difficulty int) *Blockchain {
	if difficulty < MinDifficulty {
		difficulty = MinDifficulty
	}
	return &Blockchain{difficulty: difficulty}
}

func (bc *Blockchain) GetDifficulty() int {
	return bc.difficulty
}

func (bc *Blockchain) AdjustDifficulty(actualSeconds int64) {
	target := int64(TargetBlockTimeSeconds * AdjustmentInterval)
	switch {
	case actualSeconds < target/2:
		bc.difficulty++
	case actualSeconds > target*2 && bc.difficulty > MinDifficulty:
		bc.difficulty--
	}
}

func (bc *Blockchain) IsValidNewBlock(block *block.Block) bool {
	return block.PrevHash == bc.lastHash
}

func (bc *Blockchain) UpdateWithNewBlock(block *block.Block) {
	bc.setLastHash(block.Hash)
	bc.height++
}

func (bc *Blockchain) setLastHash(hash string) {
	bc.lastHash = hash
}

func (bc *Blockchain) LastBlockHash() string {
	return bc.lastHash
}

func (bc *Blockchain) CountBlocks() int {
	return bc.height
}

func (bc *Blockchain) Mine(b *block.Block) {
	prefix := strings.Repeat("0", bc.difficulty)

	for {
		hash := b.CalculateHash()
		if strings.HasPrefix(hash, prefix) {
			b.Hash = hash
			break
		}
		b.Nonce++
	}
}

func (bc *Blockchain) IsValidPoW(b *block.Block) bool {
	prefix := strings.Repeat("0", bc.difficulty)
	return strings.HasPrefix(b.Hash, prefix)
}
