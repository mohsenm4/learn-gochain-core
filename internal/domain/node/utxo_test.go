package node

import (
	"encoding/hex"
	"testing"

	"github.com/Mohsen20031203/learn-gochain-core/internal/domain/transaction"
	"github.com/Mohsen20031203/learn-gochain-core/internal/domain/wallet"
)

// signTx fills every input with the wallet's signature and pubkey, then recomputes the tx ID.
func signTx(t *testing.T, w *wallet.Wallet, tx *transaction.Transaction) {
	t.Helper()
	sig, err := w.Sign(tx.SigningHash())
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	pub := w.PublicKeyHex()
	for i := range tx.Inputs {
		tx.Inputs[i].Signature = sig
		tx.Inputs[i].PubKey = pub
	}
	tx.ID = tx.ComputeID()
}

// seedNodeWithUTXO funds w.Address() with `amount` via a coinbase, and returns the coinbase txID.
func seedNodeWithUTXO(t *testing.T, n *Node, w *wallet.Wallet, amount uint64) string {
	t.Helper()
	cb := transaction.NewCoinbase(w.Address(), amount, 0)
	n.ApplyTx(cb)
	return cb.ID
}

func TestValidateTx_ValidSignature(t *testing.T) {
	n := NewNode("test", 1)
	owner, _ := wallet.New()
	recipient, _ := wallet.New()
	utxoID := seedNodeWithUTXO(t, n, owner, 100)

	tx := transaction.NewTransaction(
		[]transaction.TxInput{{TxID: utxoID, OutIndex: 0}},
		[]transaction.TxOutput{{Value: 50, Address: recipient.Address()}},
	)
	signTx(t, owner, tx)

	if err := n.ValidateTx(tx); err != nil {
		t.Fatalf("valid tx should pass validation, got: %v", err)
	}
}

func TestValidateTx_WrongPubKeyForUTXO(t *testing.T) {
	n := NewNode("test", 1)
	owner, _ := wallet.New()
	attacker, _ := wallet.New()
	utxoID := seedNodeWithUTXO(t, n, owner, 100)

	// Attacker uses their own pubkey/signature to try to spend owner's UTXO.
	tx := transaction.NewTransaction(
		[]transaction.TxInput{{TxID: utxoID, OutIndex: 0}},
		[]transaction.TxOutput{{Value: 50, Address: attacker.Address()}},
	)
	signTx(t, attacker, tx)

	if err := n.ValidateTx(tx); err == nil {
		t.Fatal("attacker should not be able to spend owner's UTXO with their own pubkey")
	}
}

func TestValidateTx_TamperedSignature(t *testing.T) {
	n := NewNode("test", 1)
	owner, _ := wallet.New()
	recipient, _ := wallet.New()
	utxoID := seedNodeWithUTXO(t, n, owner, 100)

	tx := transaction.NewTransaction(
		[]transaction.TxInput{{TxID: utxoID, OutIndex: 0}},
		[]transaction.TxOutput{{Value: 50, Address: recipient.Address()}},
	)
	signTx(t, owner, tx)

	// Flip a byte in the signature.
	raw, _ := hex.DecodeString(tx.Inputs[0].Signature)
	raw[5] ^= 0xff
	tx.Inputs[0].Signature = hex.EncodeToString(raw)

	if err := n.ValidateTx(tx); err == nil {
		t.Fatal("tampered signature should fail validation")
	}
}

func TestValidateTx_MissingPubKey(t *testing.T) {
	n := NewNode("test", 1)
	owner, _ := wallet.New()
	recipient, _ := wallet.New()
	utxoID := seedNodeWithUTXO(t, n, owner, 100)

	tx := transaction.NewTransaction(
		[]transaction.TxInput{{TxID: utxoID, OutIndex: 0}},
		[]transaction.TxOutput{{Value: 50, Address: recipient.Address()}},
	)
	signTx(t, owner, tx)
	tx.Inputs[0].PubKey = ""

	if err := n.ValidateTx(tx); err == nil {
		t.Fatal("missing pubkey should fail validation")
	}
}

func TestValidateTx_OutputsExceedInputs(t *testing.T) {
	n := NewNode("test", 1)
	owner, _ := wallet.New()
	recipient, _ := wallet.New()
	utxoID := seedNodeWithUTXO(t, n, owner, 100)

	tx := transaction.NewTransaction(
		[]transaction.TxInput{{TxID: utxoID, OutIndex: 0}},
		[]transaction.TxOutput{{Value: 200, Address: recipient.Address()}},
	)
	signTx(t, owner, tx)

	if err := n.ValidateTx(tx); err == nil {
		t.Fatal("outputs > inputs should fail validation")
	}
}

func TestValidateTx_DuplicateInput(t *testing.T) {
	n := NewNode("test", 1)
	owner, _ := wallet.New()
	recipient, _ := wallet.New()
	utxoID := seedNodeWithUTXO(t, n, owner, 100)

	tx := transaction.NewTransaction(
		[]transaction.TxInput{
			{TxID: utxoID, OutIndex: 0},
			{TxID: utxoID, OutIndex: 0},
		},
		[]transaction.TxOutput{{Value: 150, Address: recipient.Address()}},
	)
	signTx(t, owner, tx)

	if err := n.ValidateTx(tx); err == nil {
		t.Fatal("duplicate input should fail validation")
	}
}

func TestValidateTx_UnknownUTXO(t *testing.T) {
	n := NewNode("test", 1)
	owner, _ := wallet.New()
	recipient, _ := wallet.New()

	tx := transaction.NewTransaction(
		[]transaction.TxInput{{TxID: "nonexistent", OutIndex: 0}},
		[]transaction.TxOutput{{Value: 50, Address: recipient.Address()}},
	)
	signTx(t, owner, tx)

	if err := n.ValidateTx(tx); err == nil {
		t.Fatal("unknown UTXO should fail validation")
	}
}

func TestValidateTx_Coinbase(t *testing.T) {
	n := NewNode("test", 1)
	cb := transaction.NewCoinbase(wallet.GenesisAddress, 50, 0)

	if err := n.ValidateTx(cb); err != nil {
		t.Fatalf("coinbase should validate without signatures, got: %v", err)
	}
}

func TestValidateTx_CoinbaseWithNoOutputs(t *testing.T) {
	n := NewNode("test", 1)
	// hand-craft a malformed coinbase (no outputs)
	tx := &transaction.Transaction{}
	tx.ID = tx.ComputeID()

	if err := n.ValidateTx(tx); err == nil {
		t.Fatal("coinbase with no outputs should fail")
	}
}

func TestValidateTx_RebuildAfterTampering(t *testing.T) {
	// Demonstrates that changing outputs after signing invalidates the signature,
	// even if the attacker recomputes the tx ID.
	n := NewNode("test", 1)
	owner, _ := wallet.New()
	recipient, _ := wallet.New()
	attacker, _ := wallet.New()
	utxoID := seedNodeWithUTXO(t, n, owner, 100)

	tx := transaction.NewTransaction(
		[]transaction.TxInput{{TxID: utxoID, OutIndex: 0}},
		[]transaction.TxOutput{{Value: 50, Address: recipient.Address()}},
	)
	signTx(t, owner, tx)

	// Attacker redirects the output to their own address and recomputes the ID.
	tx.Outputs[0].Address = attacker.Address()
	tx.ID = tx.ComputeID()

	if err := n.ValidateTx(tx); err == nil {
		t.Fatal("modifying outputs after signing must invalidate the signature")
	}
}

func TestFee_NonCoinbase(t *testing.T) {
	n := NewNode("test", 1)
	owner, _ := wallet.New()
	recipient, _ := wallet.New()
	utxoID := seedNodeWithUTXO(t, n, owner, 100)

	tx := transaction.NewTransaction(
		[]transaction.TxInput{{TxID: utxoID, OutIndex: 0}},
		[]transaction.TxOutput{{Value: 70, Address: recipient.Address()}},
	)
	signTx(t, owner, tx)

	if fee := n.Fee(tx); fee != 30 {
		t.Fatalf("Fee = %d, want 30", fee)
	}
}

func TestFee_Coinbase(t *testing.T) {
	n := NewNode("test", 1)
	cb := transaction.NewCoinbase(wallet.GenesisAddress, 50, 0)
	if fee := n.Fee(cb); fee != 0 {
		t.Fatalf("coinbase fee = %d, want 0", fee)
	}
}
