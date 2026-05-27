package wallet

import (
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/pbkdf2"
)

const (
	// EntropyBytes is the number of random bytes encoded in a mnemonic.
	EntropyBytes = 16
	// MnemonicWords is the resulting word count: EntropyBytes + 1 checksum byte.
	MnemonicWords = EntropyBytes + 1
	// SeedBytes is the length of the stretched seed used to grow the HD tree.
	SeedBytes = 64
	// pbkdf2Iterations slows down brute force on partially-known mnemonics.
	pbkdf2Iterations = 2048
)

// NewMnemonic generates fresh entropy and returns its mnemonic encoding.
func NewMnemonic() (string, error) {
	entropy := make([]byte, EntropyBytes)
	if _, err := rand.Read(entropy); err != nil {
		return "", fmt.Errorf("read entropy: %w", err)
	}
	return EncodeMnemonic(entropy)
}

// EncodeMnemonic turns raw entropy into a space-separated word list with a
// one-byte checksum at the end so typos are caught at decode time.
func EncodeMnemonic(entropy []byte) (string, error) {
	if len(entropy) != EntropyBytes {
		return "", fmt.Errorf("entropy must be %d bytes, got %d", EntropyBytes, len(entropy))
	}
	payload := append([]byte{}, entropy...)
	payload = append(payload, mnemonicChecksum(entropy))

	words := make([]string, len(payload))
	for i, b := range payload {
		words[i] = wordlist[b]
	}
	return strings.Join(words, " "), nil
}

// DecodeMnemonic reverses EncodeMnemonic and verifies the checksum.
func DecodeMnemonic(mnemonic string) ([]byte, error) {
	words := strings.Fields(strings.ToLower(strings.TrimSpace(mnemonic)))
	if len(words) != MnemonicWords {
		return nil, fmt.Errorf("mnemonic must be %d words, got %d", MnemonicWords, len(words))
	}
	payload := make([]byte, len(words))
	for i, w := range words {
		idx := wordIndex(w)
		if idx < 0 {
			return nil, fmt.Errorf("unknown word %q at position %d", w, i)
		}
		payload[i] = byte(idx)
	}
	entropy, got := payload[:EntropyBytes], payload[EntropyBytes]
	if got != mnemonicChecksum(entropy) {
		return nil, errors.New("mnemonic checksum mismatch (typo?)")
	}
	return entropy, nil
}

// MnemonicToSeed stretches the mnemonic (plus optional passphrase) into the
// 64-byte seed that seeds the HD tree. Every passphrase produces a different
// valid seed — there is no "wrong" passphrase, only different wallets.
func MnemonicToSeed(mnemonic, passphrase string) []byte {
	salt := []byte("gochain-mnemonic" + passphrase)
	return pbkdf2.Key([]byte(mnemonic), salt, pbkdf2Iterations, SeedBytes, sha256.New)
}

func mnemonicChecksum(entropy []byte) byte {
	sum := sha256.Sum256(entropy)
	return sum[0]
}
