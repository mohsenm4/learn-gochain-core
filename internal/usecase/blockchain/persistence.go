package blockchain

import (
	"fmt"

	"github.com/Mohsen20031203/learn-gochain-core/internal/domain/block"
	domainchain "github.com/Mohsen20031203/learn-gochain-core/internal/domain/blockchain"
)

func (s *NodeService) saveBlock(b *block.Block) error {
	if err := s.repo.Save(LastBlockKey, b); err != nil {
		return err
	}
	if err := s.repo.Save(b.Hash, b); err != nil {
		return err
	}
	s.node.UpdateChain(*b)

	for i := range b.Transactions {
		tx := b.Transactions[i]
		s.node.ApplyTx(&tx)
		s.node.IndexTx(tx.ID, b.Index)
	}

	s.maybeAdjustDifficulty(b)
	return nil
}

func (s *NodeService) maybeAdjustDifficulty(b *block.Block) {
	if b.Index == 0 || b.Index%domainchain.AdjustmentInterval != 0 {
		return
	}
	windowStart, err := s.walkBack(b, domainchain.AdjustmentInterval)
	if err != nil || windowStart == nil {
		return
	}
	// Skip the first window: genesis has a fixed historical timestamp,
	// so elapsed would span years and produce a bogus adjustment.
	if windowStart.PrevHash == "0" {
		return
	}
	elapsed := b.Timestamp.Sub(windowStart.Timestamp).Seconds()
	before := s.node.GetChainDifficulty()
	s.node.AdjustDifficulty(int64(elapsed))
	after := s.node.GetChainDifficulty()
	if before != after {
		fmt.Printf("[difficulty] window=%d blocks elapsed=%.1fs %d->%d\n",
			domainchain.AdjustmentInterval, elapsed, before, after)
	}
}

func (s *NodeService) walkBack(from *block.Block, n int) (*block.Block, error) {
	current := from
	for i := 0; i < n; i++ {
		if current.PrevHash == "0" || current.PrevHash == "" {
			return nil, nil
		}
		prev, err := s.repo.Get(current.PrevHash)
		if err != nil || prev == nil || prev.Hash == "" {
			return nil, err
		}
		current = prev
	}
	return current, nil
}
