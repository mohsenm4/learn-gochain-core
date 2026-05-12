package utxo

import (
	"sync"

	"github.com/Mohsen20031203/learn-gochain-core/internal/domain/transaction"
)

type Key struct {
	TxID  string
	Index int
}

type Entry struct {
	Key    Key
	Output transaction.TxOutput
}

type Set struct {
	mu   sync.RWMutex
	utxo map[Key]transaction.TxOutput
}

func NewSet() *Set {
	return &Set{utxo: make(map[Key]transaction.TxOutput)}
}

// Apply spends the transaction's inputs and adds its outputs to the set.
// Caller must validate the transaction first.
func (s *Set) Apply(tx *transaction.Transaction) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, in := range tx.Inputs {
		delete(s.utxo, Key{TxID: in.TxID, Index: in.OutIndex})
	}
	for i, out := range tx.Outputs {
		s.utxo[Key{TxID: tx.ID, Index: i}] = out
	}
}

func (s *Set) Get(txID string, idx int) (transaction.TxOutput, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out, ok := s.utxo[Key{TxID: txID, Index: idx}]
	return out, ok
}

func (s *Set) BalanceOf(address string) uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var balance uint64
	for _, out := range s.utxo {
		if out.Address == address {
			balance += out.Value
		}
	}
	return balance
}

// FindSpendable returns inputs owned by address whose values sum to at
// least amount, plus the total collected. Returns total < amount if the
// address does not own enough.
func (s *Set) FindSpendable(address string, amount uint64) (refs []transaction.TxInput, total uint64) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for k, out := range s.utxo {
		if out.Address != address {
			continue
		}
		refs = append(refs, transaction.TxInput{
			TxID:      k.TxID,
			OutIndex:  k.Index,
			Signature: address,
		})
		total += out.Value
		if total >= amount {
			return refs, total
		}
	}
	return refs, total
}

func (s *Set) ListByAddress(address string) []Entry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var out []Entry
	for k, v := range s.utxo {
		if v.Address == address {
			out = append(out, Entry{Key: k, Output: v})
		}
	}
	return out
}
