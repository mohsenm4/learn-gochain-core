package blockchain

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/Mohsen20031203/learn-gochain-core/config"
	"github.com/Mohsen20031203/learn-gochain-core/internal/domain/block"
	"github.com/Mohsen20031203/learn-gochain-core/internal/domain/node"
	"github.com/Mohsen20031203/learn-gochain-core/internal/domain/transaction"
	"github.com/Mohsen20031203/learn-gochain-core/internal/infrastructure/network"
	"github.com/Mohsen20031203/learn-gochain-core/internal/infrastructure/storage/lvldb"
)

// 1. validata the tx
// 2. put the mempool
// 3. mine a new block
// 4. put the txs mompool to the new block
// 5. validate the block and txs
// 6. save new block to the repo

type NodeService struct {
	node        *node.Node
	repo        Repository
	config      config.Config
	mineTrigger chan struct{}
	gossiper    *network.TCPGossiper

	seenMu sync.Mutex
	seen   map[string]struct{}
}

func (s *NodeService) SetGossiper(g *network.TCPGossiper) {
	s.gossiper = g
}

// messageID is a content hash used to deduplicate gossiped messages so a node
// never processes or re-forwards the same payload twice.
func messageID(msg network.Message) string {
	h := sha256.New()
	h.Write([]byte(msg.Type))
	h.Write([]byte{':'})
	h.Write(msg.Data)
	return hex.EncodeToString(h.Sum(nil))
}

func (s *NodeService) markSeen(id string) bool {
	s.seenMu.Lock()
	defer s.seenMu.Unlock()
	if _, ok := s.seen[id]; ok {
		return false
	}
	s.seen[id] = struct{}{}
	return true
}

func (s *NodeService) HandleNodeMessage(msg network.Message) {
	if !s.markSeen(messageID(msg)) {
		return
	}

	switch msg.Type {
	case "block":
		var blc block.Block
		if err := json.Unmarshal(msg.Data, &blc); err != nil {
			fmt.Println("error unmarshall block from node message:", err)
			return
		}
		if !s.validataBlock(blc) {
			fmt.Println("received invalid block from peer")
			return
		}
		if err := s.saveBlock(&blc); err != nil {
			fmt.Println("error saving block from node message:", err)
			return
		}
		fmt.Println("block saved from peer:", blc.Hash)
		s.forward(msg)
	case "tx":
		var txs []transaction.Transaction
		if err := json.Unmarshal(msg.Data, &txs); err != nil {
			fmt.Println("error unmarshall txs from node message:", err)
			return
		}
		s.SubmitTransactions(txs)
		s.forward(msg)
	}
}

func (s *NodeService) forward(msg network.Message) {
	if s.gossiper == nil {
		return
	}
	s.gossiper.Gossip(msg)
}

func NewService(config config.Config) *NodeService {

	repo := lvldb.New(config.FileStoragePath)
	repo.Open()

	node := node.NewNode(config.NodeID, config.Difficulty)

	return &NodeService{
		node:        node,
		repo:        repo,
		config:      config,
		mineTrigger: make(chan struct{}),
		seen:        make(map[string]struct{}),
	}
}

func (s *NodeService) SubmitTransactions(tx []transaction.Transaction) error {

	for _, t := range tx {
		s.node.AddTransactionMempool(t)
	}
	if s.node.SizeMempool() >= s.config.BatchSize {
		select {
		case s.mineTrigger <- struct{}{}:
		default:
		}
	}
	return nil
}

func (s *NodeService) GetLastBlock() (*block.Block, error) {
	last, err := s.repo.Get(LastBlockKey)
	if err != nil {
		return nil, err
	}
	if last.Hash == "" {
		return nil, nil
	}
	return last, nil
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

func (s *NodeService) validataBlock(blc block.Block) bool {

	lastBlockHash := s.node.GetChainLastBlockHash()
	if lastBlockHash != "" {
		if !s.node.IsValidNewBlockChain(blc) {
			fmt.Println("Invalid block: previous hash does not match")
			return false
		}
	} else {
		if blc.Index != 0 {
			fmt.Println("Invalid block: genesis block index must be 0")
			return false
		}
		if !s.node.IsValidPoW(&blc) {
			fmt.Println("Invalid block: proof of work is not valid")
			return false
		}
		return true
	}

	if blc.Index < s.node.CountBlocksinChain() {
		fmt.Println("Invalid block: index is not greater than last block index")
		return false
	}

	if !s.node.IsValidPoW(&blc) {
		fmt.Println("Invalid block: proof of work is not valid")
		return false
	}

	b, err := s.repo.Get(blc.Hash)
	if err == nil && b.Hash != "" {
		fmt.Println("Invalid block: block already exists")
		return false
	}

	return true
}

func (s *NodeService) mineOnce() {
	tx2 := s.node.GetMempoolTransaction(s.config.BatchSize)
	if len(tx2) == 0 {
		return
	}

	tx := make([]transaction.Transaction, len(tx2))
	copy(tx, tx2)

	lastBlock := s.node.GetChainLastBlockHash()

	var blc *block.Block
	if lastBlock == "" {
		blc = block.NewBlock(0, tx, "0")
	} else {
		blc = block.NewBlock(s.node.CountBlocksinChain(), tx, lastBlock)
	}

	s.node.MineBlock(blc)

	if lastBlock != "" && !s.node.IsValidNewBlockChain(*blc) {
		fmt.Println("Invalid mined block")
		return
	}

	s.saveBlock(blc)
	s.gossipBlock(blc)

	if s.node.SizeMempool() == len(tx) {
		s.node.ClearMempool()
		return
	}

	for _, t := range tx {
		s.node.RemoveTransactionMempool(t)
	}
}

func (s *NodeService) gossipBlock(blc *block.Block) {
	if s.gossiper == nil {
		return
	}

	data, err := json.Marshal(blc)
	if err != nil {
		return
	}

	msg := network.Message{
		Type: "block",
		Data: data,
	}

	s.markSeen(messageID(msg))
	s.gossiper.Gossip(msg)
}

func (s *NodeService) saveBlock(b *block.Block) error {
	lastBlock := s.node.GetChainLastBlockHash()
	if lastBlock != "" {
		bl, err := s.repo.Get(lastBlock)
		if err != nil {
			return err
		}
		if err := s.repo.Save(lastBlock, bl); err != nil {
			return err
		}
	}

	if err := s.repo.Save(LastBlockKey, b); err != nil {
		return err
	}
	if err := s.repo.Save(b.Hash, b); err != nil {
		return err
	}
	s.node.UpdateChain(*b)
	return nil
}

func (s *NodeService) GetMempoolTransactions() []transaction.Transaction {
	return s.node.GetMempoolTransactions()
}

func (s *NodeService) GetChain() ([]block.Block, error) {
	var chain []block.Block

	current, err := s.repo.Get(LastBlockKey)
	if err != nil {
		return nil, err
	}
	for current.Hash != "" {
		chain = append([]block.Block{*current}, chain...) // prepend
		if current.PrevHash == "0" {
			break
		}
		current, err = s.repo.Get(current.PrevHash)
		if err != nil {
			return nil, err
		}
	}

	return chain, nil
}

func (s *NodeService) GetBlockByHash(block string) (*block.Block, error) {
	value, err := s.repo.Get(block)
	if err != nil {
		return nil, err
	}
	return value, nil
}
