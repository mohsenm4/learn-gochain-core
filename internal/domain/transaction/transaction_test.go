package transaction

import (
	"bytes"
	"testing"
)

func TestSigningHash_IgnoresSignatureAndPubKey(t *testing.T) {
	a := &Transaction{
		Inputs:  []TxInput{{TxID: "a", OutIndex: 0, Signature: "sig1", PubKey: "pk1"}},
		Outputs: []TxOutput{{Value: 10, Address: "addr"}},
		Time:    1000,
	}
	b := &Transaction{
		Inputs:  []TxInput{{TxID: "a", OutIndex: 0, Signature: "totally different", PubKey: "also different"}},
		Outputs: []TxOutput{{Value: 10, Address: "addr"}},
		Time:    1000,
	}
	if !bytes.Equal(a.SigningHash(), b.SigningHash()) {
		t.Fatal("SigningHash must be identical regardless of Signature/PubKey content")
	}
}

func TestSigningHash_ChangesWithOutputs(t *testing.T) {
	a := &Transaction{
		Inputs:  []TxInput{{TxID: "a", OutIndex: 0}},
		Outputs: []TxOutput{{Value: 10, Address: "addr"}},
		Time:    1000,
	}
	b := &Transaction{
		Inputs:  []TxInput{{TxID: "a", OutIndex: 0}},
		Outputs: []TxOutput{{Value: 11, Address: "addr"}},
		Time:    1000,
	}
	if bytes.Equal(a.SigningHash(), b.SigningHash()) {
		t.Fatal("SigningHash must differ when outputs differ")
	}
}

func TestSigningHash_ChangesWithInputs(t *testing.T) {
	a := &Transaction{
		Inputs:  []TxInput{{TxID: "a", OutIndex: 0}},
		Outputs: []TxOutput{{Value: 10, Address: "addr"}},
		Time:    1000,
	}
	b := &Transaction{
		Inputs:  []TxInput{{TxID: "b", OutIndex: 0}},
		Outputs: []TxOutput{{Value: 10, Address: "addr"}},
		Time:    1000,
	}
	if bytes.Equal(a.SigningHash(), b.SigningHash()) {
		t.Fatal("SigningHash must differ when referenced inputs differ")
	}
}

func TestComputeID_Deterministic(t *testing.T) {
	tx := &Transaction{
		Inputs:  []TxInput{{TxID: "a", OutIndex: 0, Signature: "s", PubKey: "p"}},
		Outputs: []TxOutput{{Value: 1, Address: "x"}},
		Time:    100,
	}
	if tx.ComputeID() != tx.ComputeID() {
		t.Fatal("ComputeID must be deterministic for the same tx")
	}
}

func TestNewCoinbase(t *testing.T) {
	tx := NewCoinbase("0xabc", 50, 1000)
	if !tx.IsCoinbase() {
		t.Fatal("expected IsCoinbase=true")
	}
	if tx.ID == "" {
		t.Fatal("coinbase ID should be set by NewCoinbase")
	}
	if len(tx.Outputs) != 1 {
		t.Fatalf("coinbase must have exactly one output, got %d", len(tx.Outputs))
	}
	if tx.Outputs[0].Value != 50 {
		t.Fatalf("coinbase value = %d, want 50", tx.Outputs[0].Value)
	}
	if tx.Outputs[0].Address != "0xabc" {
		t.Fatalf("coinbase address = %s, want 0xabc", tx.Outputs[0].Address)
	}
}

func TestNewTransaction_IsNotCoinbase(t *testing.T) {
	tx := NewTransaction(
		[]TxInput{{TxID: "a", OutIndex: 0}},
		[]TxOutput{{Value: 5, Address: "x"}},
	)
	if tx.IsCoinbase() {
		t.Fatal("transaction with inputs must not be classified as coinbase")
	}
	if tx.ID == "" {
		t.Fatal("transaction ID should be set by NewTransaction")
	}
}
