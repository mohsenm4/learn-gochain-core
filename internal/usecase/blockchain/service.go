package blockchain

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/Mohsen20031203/learn-gochain-core/config"
	"github.com/Mohsen20031203/learn-gochain-core/internal/domain/block"
	"github.com/Mohsen20031203/learn-gochain-core/internal/domain/node"
	"github.com/Mohsen20031203/learn-gochain-core/internal/domain/transaction"
	"github.com/Mohsen20031203/learn-gochain-core/internal/domain/utxo"
	"github.com/Mohsen20031203/learn-gochain-core/internal/domain/wallet"
	"github.com/Mohsen20031203/learn-gochain-core/internal/infrastructure/network"
	"github.com/Mohsen20031203/learn-gochain-core/internal/infrastructure/storage/lvldb"
)

// Fixed miner subsidy per block, on top of fees.
const MinerReward = uint64(50)

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

func (s *NodeService) SetGossiper(g *network.TCPGossiper) {
	s.gossiper = g
}

// Content hash used to dedupe gossiped messages.
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
		if err := s.SubmitTransactions(txs); err != nil {
			fmt.Println("error submitting incoming txs:", err)
			return
		}
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
	if len(blc.Transactions) == 0 {
		fmt.Println("Invalid block: no transactions (coinbase required)")
		return false
	}
	if !blc.Transactions[0].IsCoinbase() {
		fmt.Println("Invalid block: first transaction must be coinbase")
		return false
	}

	lastBlockHash := s.node.GetChainLastBlockHash()
	if lastBlockHash == "" {
		if blc.Index != 0 {
			fmt.Println("Invalid block: genesis block index must be 0")
			return false
		}
		return true
	}

	if !s.node.IsValidNewBlockChain(blc) {
		fmt.Println("Invalid block: previous hash does not match")
		return false
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

	coinbase := transaction.NewCoinbase(s.minerAddress, MinerReward+totalFee, time.Now().UnixNano())
	txs := append([]transaction.Transaction{*coinbase}, included...)

	lastBlock := s.node.GetChainLastBlockHash()
	if lastBlock == "" {
		return errors.New("cannot mine: genesis not initialized")
	}
	blc := block.NewBlock(s.node.CountBlocksinChain(), txs, lastBlock)
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

type TxStatus struct {
	TxID          string `json:"tx_id"`
	BlockIndex    int    `json:"block_index"`
	Confirmations int    `json:"confirmations"`
	Found         bool   `json:"found"`
}

func (s *NodeService) GetTxStatus(txID string) TxStatus {
	idx, ok := s.node.TxBlockIndex(txID)
	if !ok {
		return TxStatus{TxID: txID, Found: false}
	}
	return TxStatus{
		TxID:          txID,
		BlockIndex:    idx,
		Confirmations: s.node.Confirmations(txID),
		Found:         true,
	}
}

func (s *NodeService) GetBalance(address string) uint64 {
	return s.node.BalanceOf(address)
}

func (s *NodeService) GetUTXOs(address string) []utxo.Entry {
	return s.node.UTXOsOf(address)
}

type ChainInfo struct {
	NodeID        string `json:"node_id"`
	Blocks        int    `json:"blocks"`
	BestBlockHash string `json:"best_block_hash"`
	Difficulty    int    `json:"difficulty"`
	MempoolSize   int    `json:"mempool_size"`
}

func (s *NodeService) GetChainInfo() ChainInfo {
	return ChainInfo{
		NodeID:        s.node.GetID(),
		Blocks:        s.node.CountBlocksinChain(),
		BestBlockHash: s.node.GetChainLastBlockHash(),
		Difficulty:    s.node.GetChainDifficulty(),
		MempoolSize:   s.node.SizeMempool(),
	}
}

type NetworkInfo struct {
	NodeID     string   `json:"node_id"`
	TCPAddress string   `json:"tcp_address"`
	APIPort    string   `json:"api_port"`
	Peers      []string `json:"peers"`
	PeerCount  int      `json:"peer_count"`
}

func (s *NodeService) GetNetworkInfo() NetworkInfo {
	peers := append([]string{}, s.config.Peers...)
	return NetworkInfo{
		NodeID:     s.node.GetID(),
		TCPAddress: s.config.TCPAddress,
		APIPort:    s.config.Port,
		Peers:      peers,
		PeerCount:  len(peers),
	}
}
