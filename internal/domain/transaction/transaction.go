package transaction

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"
)

// TxInput references an unspent output owned by the sender.
// Signature is simplified: it must equal the referenced output's Address.
type TxInput struct {
	TxID      string `json:"tx_id"`
	OutIndex  int    `json:"out_index"`
	Signature string `json:"signature"`
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

// ComputeID returns a deterministic content hash of the transaction.
func (t *Transaction) ComputeID() string {
	copyTx := *t
	copyTx.ID = ""
	data, _ := json.Marshal(copyTx)
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
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
