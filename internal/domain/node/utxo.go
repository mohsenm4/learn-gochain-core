package node

import (
	"errors"
	"fmt"

	"github.com/Mohsen20031203/learn-gochain-core/internal/domain/transaction"
	"github.com/Mohsen20031203/learn-gochain-core/internal/domain/utxo"
)

// ValidateTx checks that a transaction's inputs reference existing UTXOs,
// that the signatures match, and that outputs do not exceed inputs.
// Coinbase transactions (no inputs) skip these checks but must have outputs.
func (n *Node) ValidateTx(tx *transaction.Transaction) error {
	if tx.IsCoinbase() {
		if len(tx.Outputs) == 0 {
			return errors.New("coinbase must have at least one output")
		}
		return nil
	}

	spent := make(map[utxo.Key]struct{}, len(tx.Inputs))
	var inSum, outSum uint64
	for _, in := range tx.Inputs {
		k := utxo.Key{TxID: in.TxID, Index: in.OutIndex}
		if _, dup := spent[k]; dup {
			return fmt.Errorf("input %s:%d referenced twice", in.TxID, in.OutIndex)
		}
		spent[k] = struct{}{}

		out, ok := n.utxo.Get(in.TxID, in.OutIndex)
		if !ok {
			return fmt.Errorf("unknown or already-spent utxo %s:%d", in.TxID, in.OutIndex)
		}
		if in.Signature != out.Address {
			return fmt.Errorf("invalid signature for %s:%d", in.TxID, in.OutIndex)
		}
		inSum += out.Value
	}
	for _, out := range tx.Outputs {
		outSum += out.Value
	}
	if outSum > inSum {
		return errors.New("outputs exceed inputs")
	}
	return nil
}

// Fee returns inputs minus outputs. Zero for coinbase.
func (n *Node) Fee(tx *transaction.Transaction) uint64 {
	if tx.IsCoinbase() {
		return 0
	}
	var inSum, outSum uint64
	for _, in := range tx.Inputs {
		out, ok := n.utxo.Get(in.TxID, in.OutIndex)
		if !ok {
			return 0
		}
		inSum += out.Value
	}
	for _, out := range tx.Outputs {
		outSum += out.Value
	}
	return inSum - outSum
}

func (n *Node) ApplyTx(tx *transaction.Transaction) {
	n.utxo.Apply(tx)
}

func (n *Node) BalanceOf(address string) uint64 {
	return n.utxo.BalanceOf(address)
}

func (n *Node) UTXOsOf(address string) []utxo.Entry {
	return n.utxo.ListByAddress(address)
}

func (n *Node) FindSpendable(address string, amount uint64) ([]transaction.TxInput, uint64) {
	return n.utxo.FindSpendable(address, amount)
}
