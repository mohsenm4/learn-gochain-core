package node

import (
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/Mohsen20031203/learn-gochain-core/internal/domain/transaction"
	"github.com/Mohsen20031203/learn-gochain-core/internal/domain/utxo"
	"github.com/Mohsen20031203/learn-gochain-core/internal/domain/wallet"
)

// ValidateTx enforces: inputs exist, every input's PubKey hashes to the
// referenced output's Address, every input's signature verifies against the
// transaction's SigningHash, no duplicate inputs, outputs<=inputs.
// Coinbase txs are exempt (no inputs).
func (n *Node) ValidateTx(tx *transaction.Transaction) error {
	for i, out := range tx.Outputs {
		if err := wallet.ValidateAddress(out.Address); err != nil {
			return fmt.Errorf("output %d: %w", i, err)
		}
	}
	if tx.IsCoinbase() {
		if len(tx.Outputs) == 0 {
			return errors.New("coinbase must have at least one output")
		}
		return nil
	}

	spent := make(map[utxo.Key]struct{}, len(tx.Inputs))
	var inSum, outSum uint64

	sigHashHex := hex.EncodeToString(tx.SigningHash())

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

		if in.PubKey == "" {
			return fmt.Errorf("input %s:%d missing pubkey", in.TxID, in.OutIndex)
		}
		pubBytes, err := hex.DecodeString(in.PubKey)
		if err != nil {
			return fmt.Errorf("input %s:%d invalid pubkey hex: %w", in.TxID, in.OutIndex, err)
		}
		if wallet.DeriveAddress(pubBytes) != out.Address {
			return fmt.Errorf("input %s:%d pubkey does not match output address", in.TxID, in.OutIndex)
		}

		ok2, err := wallet.Verify(in.PubKey, sigHashHex, in.Signature)
		if err != nil {
			return fmt.Errorf("input %s:%d signature error: %w", in.TxID, in.OutIndex, err)
		}
		if !ok2 {
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

// Inputs minus outputs; zero for coinbase.
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
