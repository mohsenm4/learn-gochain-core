package wallet

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os"
)

// Branch indices used at the first level under the master key.
// Mirrors BIP44's receive/change split (`m/.../0` and `m/.../1`).
const (
	BranchReceive uint32 = 0
	BranchChange  uint32 = 1
)

// hdHMACKey is the HMAC key used to turn a seed into the master key/chain code.
// BIP32 hardcodes "Bitcoin seed"; we use our own constant so derived keys
// can't be confused with real Bitcoin keys.
var hdHMACKey = []byte("GoChain HD seed")

// HDWallet is a hierarchical deterministic wallet: one mnemonic on disk grows
// into a whole tree of keys. Receive and change addresses live on separate
// branches so the bookkeeping stays clean.
type HDWallet struct {
	mnemonic        string
	masterPriv      *big.Int
	masterChainCode []byte
	receiveIdx      uint32
	changeIdx       uint32
}

// NewHDWallet creates a wallet from fresh entropy. The returned mnemonic is
// the only thing the caller needs to back up.
func NewHDWallet() (*HDWallet, string, error) {
	mnemonic, err := NewMnemonic()
	if err != nil {
		return nil, "", err
	}
	h, err := HDFromMnemonic(mnemonic, "")
	if err != nil {
		return nil, "", err
	}
	return h, mnemonic, nil
}

// HDFromMnemonic rebuilds the wallet from a mnemonic and optional passphrase.
// Every passphrase produces a different valid wallet — there is no "wrong"
// passphrase, just a different tree of keys.
func HDFromMnemonic(mnemonic, passphrase string) (*HDWallet, error) {
	if _, err := DecodeMnemonic(mnemonic); err != nil {
		return nil, fmt.Errorf("mnemonic: %w", err)
	}
	seed := MnemonicToSeed(mnemonic, passphrase)
	priv, chain := masterFromSeed(seed)
	return &HDWallet{
		mnemonic:        mnemonic,
		masterPriv:      priv,
		masterChainCode: chain,
	}, nil
}

// Mnemonic returns the recovery phrase that was used to build this wallet.
func (h *HDWallet) Mnemonic() string { return h.mnemonic }

// NextReceive derives and returns the next address on the receive branch.
// The internal counter advances so the same address is never handed out twice.
func (h *HDWallet) NextReceive() (*Wallet, error) {
	w, err := h.deriveLeaf(BranchReceive, h.receiveIdx)
	if err != nil {
		return nil, err
	}
	h.receiveIdx++
	return w, nil
}

// NextChange derives and returns the next change address.
func (h *HDWallet) NextChange() (*Wallet, error) {
	w, err := h.deriveLeaf(BranchChange, h.changeIdx)
	if err != nil {
		return nil, err
	}
	h.changeIdx++
	return w, nil
}

// DeriveReceive derives a specific receive address without advancing the
// counter. Useful for re-deriving a known address.
func (h *HDWallet) DeriveReceive(index uint32) (*Wallet, error) {
	return h.deriveLeaf(BranchReceive, index)
}

// DeriveChange derives a specific change address without advancing the counter.
func (h *HDWallet) DeriveChange(index uint32) (*Wallet, error) {
	return h.deriveLeaf(BranchChange, index)
}

// XPub returns a read-only view that can derive child addresses but cannot
// sign transactions. Safe to publish to a web server or untrusted machine.
func (h *HDWallet) XPub() *XPub {
	pub := scalarBaseMult(h.masterPriv)
	return &XPub{
		pub:       pub,
		chainCode: append([]byte{}, h.masterChainCode...),
	}
}

// deriveLeaf walks two levels: m -> branch -> index.
func (h *HDWallet) deriveLeaf(branch, index uint32) (*Wallet, error) {
	branchPriv, branchChain, err := ckdPriv(h.masterPriv, h.masterChainCode, branch)
	if err != nil {
		return nil, err
	}
	leafPriv, _, err := ckdPriv(branchPriv, branchChain, index)
	if err != nil {
		return nil, err
	}
	return fromScalar(leafPriv), nil
}

// XPub is a public-key + chain code pair. It can derive child public keys
// (and therefore addresses) without ever touching a private key.
type XPub struct {
	pub       *ecdsa.PublicKey
	chainCode []byte
}

// DeriveReceive returns the address for receive branch at the given index.
// Equivalent to the matching HDWallet.DeriveReceive on the wallet's xprv side.
func (x *XPub) DeriveReceive(index uint32) (string, error) {
	return x.deriveAddress(BranchReceive, index)
}

// DeriveChange returns the change address at the given index.
func (x *XPub) DeriveChange(index uint32) (string, error) {
	return x.deriveAddress(BranchChange, index)
}

// Encode serializes an xpub as hex-prefixed concatenation of pub + chain code.
func (x *XPub) Encode() string {
	pubBytes := marshalPub(x.pub)
	out := make([]byte, 0, len(pubBytes)+len(x.chainCode))
	out = append(out, pubBytes...)
	out = append(out, x.chainCode...)
	return "xpub" + hex.EncodeToString(out)
}

// DecodeXPub parses an xpub produced by Encode.
func DecodeXPub(s string) (*XPub, error) {
	if len(s) < 4 || s[:4] != "xpub" {
		return nil, errors.New("xpub must start with 'xpub'")
	}
	raw, err := hex.DecodeString(s[4:])
	if err != nil {
		return nil, fmt.Errorf("xpub hex: %w", err)
	}
	if len(raw) < 33+32 {
		return nil, fmt.Errorf("xpub too short: %d bytes", len(raw))
	}
	pub, err := unmarshalPub(raw[:33])
	if err != nil {
		return nil, fmt.Errorf("xpub pubkey: %w", err)
	}
	return &XPub{
		pub:       pub,
		chainCode: append([]byte{}, raw[33:]...),
	}, nil
}

func (x *XPub) deriveAddress(branch, index uint32) (string, error) {
	branchPub, branchChain, err := ckdPub(x.pub, x.chainCode, branch)
	if err != nil {
		return "", err
	}
	leafPub, _, err := ckdPub(branchPub, branchChain, index)
	if err != nil {
		return "", err
	}
	return DeriveAddress(marshalPub(leafPub)), nil
}

// masterFromSeed splits the seed-HMAC output into master key and chain code,
// matching the BIP32 shape.
func masterFromSeed(seed []byte) (*big.Int, []byte) {
	mac := hmac.New(sha512.New, hdHMACKey)
	mac.Write(seed)
	out := mac.Sum(nil)

	curveN := elliptic.P256().Params().N
	priv := new(big.Int).SetBytes(out[:32])
	priv.Mod(priv, curveN)
	if priv.Sign() == 0 {
		priv.SetInt64(1)
	}
	chain := append([]byte{}, out[32:]...)
	return priv, chain
}

// ckdPriv performs child key derivation from a parent private key. The xpub
// side runs ckdPub with the same inputs and lands on the matching child.
func ckdPriv(parentPriv *big.Int, parentChain []byte, index uint32) (*big.Int, []byte, error) {
	parentPub := scalarBaseMult(parentPriv)
	pubBytes := marshalPub(parentPub)

	il, ir, err := hmacChain(parentChain, pubBytes, index)
	if err != nil {
		return nil, nil, err
	}

	curveN := elliptic.P256().Params().N
	child := new(big.Int).Add(il, parentPriv)
	child.Mod(child, curveN)
	if child.Sign() == 0 {
		return nil, nil, errors.New("derived zero child key (try next index)")
	}
	return child, ir, nil
}

// ckdPub mirrors ckdPriv on the public side: pub_child = pub_parent + IL*G.
func ckdPub(parentPub *ecdsa.PublicKey, parentChain []byte, index uint32) (*ecdsa.PublicKey, []byte, error) {
	pubBytes := marshalPub(parentPub)
	il, ir, err := hmacChain(parentChain, pubBytes, index)
	if err != nil {
		return nil, nil, err
	}
	curve := elliptic.P256()
	ilGx, ilGy := curve.ScalarBaseMult(il.Bytes())
	childX, childY := curve.Add(parentPub.X, parentPub.Y, ilGx, ilGy)
	if childX.Sign() == 0 && childY.Sign() == 0 {
		return nil, nil, errors.New("derived point at infinity (try next index)")
	}
	return &ecdsa.PublicKey{Curve: curve, X: childX, Y: childY}, ir, nil
}

// hmacChain computes I = HMAC-SHA512(chain, pub || index) and returns the
// halves used by ckdPriv/ckdPub.
func hmacChain(chain, parentPub []byte, index uint32) (*big.Int, []byte, error) {
	mac := hmac.New(sha512.New, chain)
	mac.Write(parentPub)
	var idxBuf [4]byte
	binary.BigEndian.PutUint32(idxBuf[:], index)
	mac.Write(idxBuf[:])
	out := mac.Sum(nil)

	il := new(big.Int).SetBytes(out[:32])
	curveN := elliptic.P256().Params().N
	if il.Cmp(curveN) >= 0 {
		return nil, nil, errors.New("IL >= N (try next index)")
	}
	return il, append([]byte{}, out[32:]...), nil
}

func scalarBaseMult(d *big.Int) *ecdsa.PublicKey {
	curve := elliptic.P256()
	x, y := curve.ScalarBaseMult(d.Bytes())
	return &ecdsa.PublicKey{Curve: curve, X: x, Y: y}
}

func fromScalar(d *big.Int) *Wallet {
	curve := elliptic.P256()
	x, y := curve.ScalarBaseMult(d.Bytes())
	priv := &ecdsa.PrivateKey{
		D: new(big.Int).Set(d),
	}
	priv.PublicKey.Curve = curve
	priv.PublicKey.X = x
	priv.PublicKey.Y = y
	priv.Curve = curve
	return &Wallet{priv: priv}
}

// hdFile is the on-disk JSON layout for an HD wallet. Only the mnemonic and
// the next-unused indices are persisted. The passphrase (if any) is never
// stored — losing it means losing the funds.
type hdFile struct {
	Mnemonic     string `json:"mnemonic"`
	ReceiveIndex uint32 `json:"receive_index"`
	ChangeIndex  uint32 `json:"change_index"`
}

// SaveHD writes the wallet to disk at mode 0600. The mnemonic lands in plain
// text, so the file is as sensitive as the words written on paper.
func (h *HDWallet) Save(path string) error {
	data := hdFile{
		Mnemonic:     h.mnemonic,
		ReceiveIndex: h.receiveIdx,
		ChangeIndex:  h.changeIdx,
	}
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0600)
}

// LoadHDWallet reads a file written by HDWallet.Save. If the wallet was
// created with a passphrase, the same passphrase must be supplied again or a
// different (and therefore empty) wallet will be reconstructed.
func LoadHDWallet(path, passphrase string) (*HDWallet, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var data hdFile
	if err := json.Unmarshal(b, &data); err != nil {
		return nil, fmt.Errorf("parse hd wallet file: %w", err)
	}
	h, err := HDFromMnemonic(data.Mnemonic, passphrase)
	if err != nil {
		return nil, err
	}
	h.receiveIdx = data.ReceiveIndex
	h.changeIdx = data.ChangeIndex
	return h, nil
}
