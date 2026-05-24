package wallet

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os"

	"github.com/Mohsen20031203/learn-gochain-core/internal/domain/transaction"
	"golang.org/x/crypto/ripemd160"
)

type SpendableUTXO struct {
	TxID     string
	OutIndex int
	Value    uint64
}

// Wallet holds an ECDSA P-256 keypair. Addresses are derived from the public key.
type Wallet struct {
	priv *ecdsa.PrivateKey
}

// New generates a fresh wallet.
func New() (*Wallet, error) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	return &Wallet{priv: priv}, nil
}

// PublicKeyBytes returns the public key as 64 raw bytes (X || Y).
func (w *Wallet) PublicKeyBytes() []byte {
	return marshalPub(&w.priv.PublicKey)
}

// PublicKeyHex is the hex encoding of PublicKeyBytes.
func (w *Wallet) PublicKeyHex() string {
	return hex.EncodeToString(w.PublicKeyBytes())
}

// Address returns the hex address derived from the public key.
func (w *Wallet) Address() string {
	return DeriveAddress(w.PublicKeyBytes())
}

// Picks UTXOs to cover amount+fee, returns the leftover as a change output.
func (w *Wallet) BuildAndSignTransaction(
	utxos []SpendableUTXO,
	toAddress string,
	amount uint64,
	fee uint64,
) (*transaction.Transaction, error) {
	if amount == 0 {
		return nil, errors.New("amount must be > 0")
	}
	if err := ValidateAddress(toAddress); err != nil {
		return nil, fmt.Errorf("invalid recipient address: %w", err)
	}

	need := amount + fee
	var picked []SpendableUTXO
	var gathered uint64
	for _, u := range utxos {
		picked = append(picked, u)
		gathered += u.Value
		if gathered >= need {
			break
		}
	}
	if gathered < need {
		return nil, fmt.Errorf("insufficient funds: have %d, need %d", gathered, need)
	}

	inputs := make([]transaction.TxInput, len(picked))
	for i, u := range picked {
		inputs[i] = transaction.TxInput{TxID: u.TxID, OutIndex: u.OutIndex}
	}

	outputs := []transaction.TxOutput{
		{Value: amount, Address: toAddress},
	}
	if change := gathered - need; change > 0 {
		outputs = append(outputs, transaction.TxOutput{
			Value:   change,
			Address: w.Address(),
		})
	}

	tx := transaction.NewTransaction(inputs, outputs)

	sigHash := tx.SigningHash()
	pubHex := w.PublicKeyHex()
	sig, err := w.Sign(sigHash)
	if err != nil {
		return nil, fmt.Errorf("sign: %w", err)
	}
	for i := range tx.Inputs {
		tx.Inputs[i].Signature = sig
		tx.Inputs[i].PubKey = pubHex
	}
	tx.ID = tx.ComputeID()
	return tx, nil
}

// Sign returns a hex-encoded signature of the given hash.
func (w *Wallet) Sign(hash []byte) (string, error) {
	r, s, err := ecdsa.Sign(rand.Reader, w.priv, hash)
	if err != nil {
		return "", err
	}
	rb := r.Bytes()
	sb := s.Bytes()
	sig := make([]byte, 64)
	copy(sig[32-len(rb):32], rb)
	copy(sig[64-len(sb):], sb)
	return hex.EncodeToString(sig), nil
}

// GenesisAddress has no private key and bypasses checksum validation.
const GenesisAddress = "genesis"

// Canonicalize to compressed form so address is a function of the EC point.
func DeriveAddress(pubKeyBytes []byte) string {
	canonical, err := canonicalCompressedPub(pubKeyBytes)
	if err != nil {
		return ""
	}
	return "0x" + hex.EncodeToString(addChecksum(hash160(canonical)))
}

func canonicalCompressedPub(b []byte) ([]byte, error) {
	pub, err := unmarshalPub(b)
	if err != nil {
		return nil, err
	}
	return elliptic.MarshalCompressed(pub.Curve, pub.X, pub.Y), nil
}

// HASH160 = ripemd160(sha256(data)). 20 bytes out.
func hash160(data []byte) []byte {
	first := sha256.Sum256(data)
	r := ripemd160.New()
	r.Write(first[:])
	return r.Sum(nil)
}

func ValidateAddress(addr string) error {
	if addr == GenesisAddress {
		return nil
	}
	if len(addr) != 50 || addr[:2] != "0x" {
		return fmt.Errorf("address must be 0x + 48 hex chars, got %q", addr)
	}
	raw, err := hex.DecodeString(addr[2:])
	if err != nil {
		return fmt.Errorf("address hex decode: %w", err)
	}
	if len(raw) != 24 {
		return fmt.Errorf("address payload must be 24 bytes, got %d", len(raw))
	}
	payload, gotSum := raw[:20], raw[20:]
	wantSum := checksum(payload)
	if !bytesEqual(gotSum, wantSum) {
		return errors.New("address checksum mismatch (typo?)")
	}
	return nil
}

func addChecksum(payload []byte) []byte {
	out := make([]byte, 0, len(payload)+4)
	out = append(out, payload...)
	out = append(out, checksum(payload)...)
	return out
}

func checksum(payload []byte) []byte {
	first := sha256.Sum256(payload)
	second := sha256.Sum256(first[:])
	return second[:4]
}

func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// Verify checks an ECDSA signature against the given hash and public key (all hex).
func Verify(pubKeyHex, hashHex, sigHex string) (bool, error) {
	pubBytes, err := hex.DecodeString(pubKeyHex)
	if err != nil {
		return false, fmt.Errorf("decode pubkey: %w", err)
	}
	pub, err := unmarshalPub(pubBytes)
	if err != nil {
		return false, err
	}
	hash, err := hex.DecodeString(hashHex)
	if err != nil {
		return false, fmt.Errorf("decode hash: %w", err)
	}
	sig, err := hex.DecodeString(sigHex)
	if err != nil {
		return false, fmt.Errorf("decode signature: %w", err)
	}
	if len(sig) != 64 {
		return false, fmt.Errorf("signature must be 64 bytes, got %d", len(sig))
	}
	r := new(big.Int).SetBytes(sig[:32])
	s := new(big.Int).SetBytes(sig[32:])
	return ecdsa.Verify(pub, hash, r, s), nil
}

// walletFile is the on-disk JSON layout.
type walletFile struct {
	PrivateKey string `json:"private_key"`
	PublicKey  string `json:"public_key"`
	Address    string `json:"address"`
}

// Save writes the wallet to disk with mode 0600.
func (w *Wallet) Save(path string) error {
	data := walletFile{
		PrivateKey: hex.EncodeToString(w.priv.D.Bytes()),
		PublicKey:  w.PublicKeyHex(),
		Address:    w.Address(),
	}
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0600)
}

// Load reads a wallet from a path written by Save.
func Load(path string) (*Wallet, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var data walletFile
	if err := json.Unmarshal(b, &data); err != nil {
		return nil, fmt.Errorf("parse wallet file: %w", err)
	}
	d, err := hex.DecodeString(data.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("decode private key: %w", err)
	}
	priv := new(ecdsa.PrivateKey)
	priv.Curve = elliptic.P256()
	priv.D = new(big.Int).SetBytes(d)
	priv.PublicKey.Curve = elliptic.P256()
	priv.PublicKey.X, priv.PublicKey.Y = priv.Curve.ScalarBaseMult(d)
	return &Wallet{priv: priv}, nil
}

// LoadOrCreate loads the wallet at path, or creates and saves a new one if missing.
// The bool indicates whether a new wallet was created.
func LoadOrCreate(path string) (*Wallet, bool, error) {
	if _, err := os.Stat(path); err == nil {
		w, err := Load(path)
		if err != nil {
			return nil, false, err
		}
		return w, false, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, false, err
	}
	w, err := New()
	if err != nil {
		return nil, false, err
	}
	if err := w.Save(path); err != nil {
		return nil, false, err
	}
	return w, true, nil
}

func marshalPub(pub *ecdsa.PublicKey) []byte {
	return elliptic.MarshalCompressed(pub.Curve, pub.X, pub.Y)
}

// Accepts 33-byte compressed, 64-byte legacy X||Y, or 65-byte uncompressed.
func unmarshalPub(b []byte) (*ecdsa.PublicKey, error) {
	curve := elliptic.P256()
	switch len(b) {
	case 33:
		x, y := elliptic.UnmarshalCompressed(curve, b)
		if x == nil {
			return nil, errors.New("invalid compressed public key")
		}
		return &ecdsa.PublicKey{Curve: curve, X: x, Y: y}, nil
	case 64:
		x := new(big.Int).SetBytes(b[:32])
		y := new(big.Int).SetBytes(b[32:])
		pub := &ecdsa.PublicKey{Curve: curve, X: x, Y: y}
		if !pub.Curve.IsOnCurve(x, y) {
			return nil, errors.New("public key not on curve")
		}
		return pub, nil
	case 65:
		x, y := elliptic.Unmarshal(curve, b)
		if x == nil {
			return nil, errors.New("invalid uncompressed public key")
		}
		return &ecdsa.PublicKey{Curve: curve, X: x, Y: y}, nil
	default:
		return nil, fmt.Errorf("public key must be 33, 64, or 65 bytes, got %d", len(b))
	}
}
