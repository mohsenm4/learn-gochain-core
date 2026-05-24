package transaction

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"
)

// TxInput spends a previous output. Signature is hex(ECDSA(SigningHash)) and PubKey
// is hex(X||Y) of the spender's public key. PubKey, when hashed, must equal the
// referenced output's Address.
type TxInput struct {
	TxID      string `json:"tx_id"`
	OutIndex  int    `json:"out_index"`
	Signature string `json:"signature"`
	PubKey    string `json:"pub_key"`
}

type TxOutput struct {
	Value   uint64 `json:"value"`
	Address string `json:"address"`
}

type Transaction struct {
	ID      string     `json:"id"`
	Inputs  []TxInput  `json:"inputs"`
	Outputs []TxOutput `json:"outputs"`
	Time    int64      `json:"time"`
}

func (t *Transaction) IsCoinbase() bool {
	return len(t.Inputs) == 0
}

// ComputeID hashes the entire transaction (signatures included) with the ID field cleared.
func (t *Transaction) ComputeID() string {
	copyTx := *t
	copyTx.ID = ""
	data, _ := json.Marshal(copyTx)
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

// SigningHash is what every input signs: SHA-256 of the transaction with ID,
// all input signatures, and all input pubkeys cleared. So all signers see the
// same hash regardless of who signs first.
func (t *Transaction) SigningHash() []byte {
	copyTx := *t
	copyTx.ID = ""
	copyTx.Inputs = make([]TxInput, len(t.Inputs))
	for i, in := range t.Inputs {
		copyTx.Inputs[i] = TxInput{
			TxID:     in.TxID,
			OutIndex: in.OutIndex,
		}
	}
	data, _ := json.Marshal(copyTx)
	h := sha256.Sum256(data)
	return h[:]
}

func NewCoinbase(toAddress string, reward uint64, ts int64) *Transaction {
	tx := &Transaction{
		Outputs: []TxOutput{{Value: reward, Address: toAddress}},
		Time:    ts,
	}
	tx.ID = tx.ComputeID()
	return tx
}

func NewTransaction(inputs []TxInput, outputs []TxOutput) *Transaction {
	tx := &Transaction{
		Inputs:  inputs,
		Outputs: outputs,
		Time:    time.Now().UnixNano(),
	}
	tx.ID = tx.ComputeID()
	return tx
}
