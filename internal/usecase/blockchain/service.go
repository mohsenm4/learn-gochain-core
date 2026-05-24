package blockchain

import (
	"fmt"
	"sync"

	"github.com/Mohsen20031203/learn-gochain-core/config"
	"github.com/Mohsen20031203/learn-gochain-core/internal/domain/block"
	"github.com/Mohsen20031203/learn-gochain-core/internal/domain/node"
	"github.com/Mohsen20031203/learn-gochain-core/internal/domain/transaction"
	"github.com/Mohsen20031203/learn-gochain-core/internal/domain/wallet"
	"github.com/Mohsen20031203/learn-gochain-core/internal/infrastructure/network"
	"github.com/Mohsen20031203/learn-gochain-core/internal/infrastructure/storage/lvldb"
)

type NodeService struct {
	node        *node.Node
	repo        Repository
	config      config.Config
	mineTrigger chan struct{}
	gossiper    *network.TCPGossiper

	minerWallet  *wallet.Wallet
	minerAddress string

	seenMu sync.Mutex
	seen   map[string]struct{}
}

func NewService(config config.Config) *NodeService {

	repo := lvldb.New(config.FileStoragePath)
	if err := repo.Open(); err != nil {
		panic(fmt.Errorf("open storage at %q: %w", config.FileStoragePath, err))
	}

	minerWallet, created, err := wallet.LoadOrCreate(config.MinerWalletPath)
	if err != nil {
		panic(fmt.Errorf("open miner wallet at %q: %w", config.MinerWalletPath, err))
	}
	if created {
		fmt.Println("[miner] new wallet created at:", config.MinerWalletPath)
	}
	fmt.Println("[miner] address:", minerWallet.Address())

	node := node.NewNode(config.NodeID, config.Difficulty)

	return &NodeService{
		node:         node,
		repo:         repo,
		config:       config,
		mineTrigger:  make(chan struct{}),
		seen:         make(map[string]struct{}),
		minerWallet:  minerWallet,
		minerAddress: minerWallet.Address(),
	}
}

// MinerAddress returns the address used for coinbase outputs.
func (s *NodeService) MinerAddress() string {
	return s.minerAddress
}

// Ensures the chain has a genesis; safe to call on fresh or restarted nodes.
func (s *NodeService) BootstrapGenesis() error {
	if s.node.GetChainLastBlockHash() != "" {
		return nil
	}
	last, err := s.repo.Get(LastBlockKey)
	if err == nil && last != nil && last.Hash != "" {
		return s.rehydrateFromRepo()
	}
	genesis := block.NewGenesis()
	return s.saveBlock(genesis)
}

// Replays persisted blocks oldest-to-newest into UTXO and tx index.
func (s *NodeService) rehydrateFromRepo() error {
	chain, err := s.GetChain()
	if err != nil {
		return err
	}
	for i := range chain {
		b := chain[i]
		for j := range b.Transactions {
			tx := b.Transactions[j]
			s.node.ApplyTx(&tx)
			s.node.IndexTx(tx.ID, b.Index)
		}
		s.node.UpdateChain(b)
	}
	return nil
}

func (s *NodeService) SubmitTransactions(txs []transaction.Transaction) error {
	for i := range txs {
		tx := txs[i]
		tx.ID = tx.ComputeID()
		if s.node.HasTransactionMempool(tx.ID) {
			continue
		}
		if err := s.node.ValidateTx(&tx); err != nil {
			return fmt.Errorf("invalid transaction %s: %w", tx.ID, err)
		}
		s.node.AddTransactionMempool(tx)
	}
	if s.node.SizeMempool() >= s.config.BatchSize {
		select {
		case s.mineTrigger <- struct{}{}:
		default:
		}
	}
	return nil
}
