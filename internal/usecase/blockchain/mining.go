package blockchain

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Mohsen20031203/learn-gochain-core/internal/domain/block"
	"github.com/Mohsen20031203/learn-gochain-core/internal/domain/transaction"
)

// Subsidy halves every HalvingInterval blocks; reward eventually decays to 0.
const (
	InitialReward   = uint64(50)
	HalvingInterval = 10
)

func blockReward(height int) uint64 {
	halvings := height / HalvingInterval
	if halvings >= 64 {
		return 0
	}
	return InitialReward >> halvings
}

func (s *NodeService) StartMiner(ctx context.Context) {
	go func() {
		for {
			select {
			case <-s.mineTrigger:
				for {
					if s.node.SizeMempool() < s.config.BatchSize {
						break
					}
					s.mineOnce()
				}
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (s *NodeService) mineOnce() {
	_ = s.mineBlock(false)
}

// ManualMine mines a block immediately, even if the mempool is empty.
// Useful for bootstrapping initial miner funds and for manual control during learning.
func (s *NodeService) ManualMine() error {
	return s.mineBlock(true)
}

func (s *NodeService) mineBlock(forced bool) error {
	candidates := s.node.GetMempoolTransaction(s.config.BatchSize)
	if len(candidates) == 0 && !forced {
		return nil
	}

	var included []transaction.Transaction
	var dropped []transaction.Transaction
	var totalFee uint64
	for _, tx := range candidates {
		if err := s.node.ValidateTx(&tx); err != nil {
			dropped = append(dropped, tx)
			continue
		}
		totalFee += s.node.Fee(&tx)
		included = append(included, tx)
	}

	if len(included) == 0 && !forced {
		for _, tx := range dropped {
			s.node.RemoveTransactionMempool(tx)
		}
		return nil
	}

	height := s.node.CountBlocksinChain()
	subsidy := blockReward(height)
	coinbase := transaction.NewCoinbase(s.minerAddress, subsidy+totalFee, time.Now().UnixNano())
	txs := append([]transaction.Transaction{*coinbase}, included...)

	lastBlock := s.node.GetChainLastBlockHash()
	if lastBlock == "" {
		return errors.New("cannot mine: genesis not initialized")
	}
	fmt.Printf("[miner] block %d subsidy=%d fees=%d total=%d\n", height, subsidy, totalFee, subsidy+totalFee)
	blc := block.NewBlock(height, txs, lastBlock)
	s.node.MineBlock(blc)

	if !s.node.IsValidNewBlockChain(*blc) {
		return errors.New("invalid mined block")
	}

	if err := s.saveBlock(blc); err != nil {
		return fmt.Errorf("save mined block: %w", err)
	}
	s.gossipBlock(blc)

	for _, tx := range included {
		s.node.RemoveTransactionMempool(tx)
	}
	for _, tx := range dropped {
		s.node.RemoveTransactionMempool(tx)
	}
	return nil
}
