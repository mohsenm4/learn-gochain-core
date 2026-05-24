package blockchain

import (
	"github.com/Mohsen20031203/learn-gochain-core/internal/domain/block"
	"github.com/Mohsen20031203/learn-gochain-core/internal/domain/transaction"
	"github.com/Mohsen20031203/learn-gochain-core/internal/domain/utxo"
)

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

func (s *NodeService) GetMempoolTransactions() []transaction.Transaction {
	return s.node.GetMempoolTransactions()
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
	var peers []string
	if s.gossiper != nil {
		peers = s.gossiper.Peers()
	} else {
		peers = append([]string{}, s.config.Peers...)
	}
	return NetworkInfo{
		NodeID:     s.node.GetID(),
		TCPAddress: s.config.TCPAddress,
		APIPort:    s.config.Port,
		Peers:      peers,
		PeerCount:  len(peers),
	}
}
