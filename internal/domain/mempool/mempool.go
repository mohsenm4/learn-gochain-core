package mempool

import "github.com/Mohsen20031203/learn-gochain-core/internal/domain/transaction"

type Mempool struct {
	transactions []transaction.Transaction
}

// NewMempool creates and returns a new Mempool instance
func NewMempool() *Mempool {
	return &Mempool{
		transactions: []transaction.Transaction{},
	}
}

// AddTransaction adds a new transaction to the mempool
func (mp *Mempool) AddTransaction(tx transaction.Transaction) {
	mp.transactions = append(mp.transactions, tx)
}

// GetTransactions returns all transactions currently in the mempool
func (mp *Mempool) GetTransactions() []transaction.Transaction {
	return mp.transactions
}

// HasTransaction checks if a transaction with the given ID exists in the mempool
func (mp *Mempool) HasTransaction(txID string) bool {
	for _, tx := range mp.transactions {
		if tx.ID == txID {
			return true
		}
	}
	return false
}

// Size returns the number of transactions in the mempool
func (mp *Mempool) Size() int {
	return len(mp.transactions)
}

func (mp *Mempool) RemoveTransaction(tx transaction.Transaction) {
	for i, t := range mp.transactions {
		if t.ID == tx.ID {
			mp.transactions = append(mp.transactions[:i], mp.transactions[i+1:]...)
			return
		}
	}
}

// Clear removes all transactions from the mempool
func (mp *Mempool) Clear() {
	mp.transactions = []transaction.Transaction{}
}

func (mp *Mempool) GetTransactionsCount(count int) []transaction.Transaction {
	if count >= len(mp.transactions) {
		return mp.transactions
	}
	return mp.transactions[:count]
}
