package wallet

import (
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNew_UniqueAddresses(t *testing.T) {
	w1, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	w2, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if w1.Address() == w2.Address() {
		t.Fatal("two freshly generated wallets must have different addresses")
	}
}

func TestSignVerify_Roundtrip(t *testing.T) {
	w, _ := New()
	hash := bytes32()
	sig, err := w.Sign(hash)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	ok, err := Verify(w.PublicKeyHex(), hex.EncodeToString(hash), sig)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if !ok {
		t.Fatal("valid signature failed to verify")
	}
}

func TestVerify_FailsWithWrongHash(t *testing.T) {
	w, _ := New()
	hash := bytes32()
	sig, _ := w.Sign(hash)
	wrong := bytes32()
	wrong[0] ^= 0xff
	ok, _ := Verify(w.PublicKeyHex(), hex.EncodeToString(wrong), sig)
	if ok {
		t.Fatal("verify should fail for a different hash")
	}
}

func TestVerify_FailsWithWrongPubKey(t *testing.T) {
	w1, _ := New()
	w2, _ := New()
	hash := bytes32()
	sig, _ := w1.Sign(hash)
	ok, _ := Verify(w2.PublicKeyHex(), hex.EncodeToString(hash), sig)
	if ok {
		t.Fatal("verify should fail when using a different pubkey")
	}
}

func TestVerify_FailsWithTamperedSignature(t *testing.T) {
	w, _ := New()
	hash := bytes32()
	sig, _ := w.Sign(hash)
	raw, _ := hex.DecodeString(sig)
	raw[0] ^= 0xff
	tampered := hex.EncodeToString(raw)
	ok, _ := Verify(w.PublicKeyHex(), hex.EncodeToString(hash), tampered)
	if ok {
		t.Fatal("verify should fail for a tampered signature")
	}
}

func TestAddress_Deterministic(t *testing.T) {
	w, _ := New()
	if w.Address() != w.Address() {
		t.Fatal("Address must be deterministic for the same wallet")
	}
}

func TestAddress_Format(t *testing.T) {
	w, _ := New()
	addr := w.Address()
	if !strings.HasPrefix(addr, "0x") {
		t.Fatalf("address must start with 0x, got %s", addr)
	}
	// 0x + 20-byte HASH160 + 4-byte checksum = 50 chars.
	if len(addr) != 50 {
		t.Fatalf("address must be 50 chars (0x + 48 hex), got %d (%s)", len(addr), addr)
	}
	if err := ValidateAddress(addr); err != nil {
		t.Fatalf("derived address must validate: %v", err)
	}
}

func TestValidateAddress_RejectsTypo(t *testing.T) {
	w, _ := New()
	addr := w.Address()
	tampered := []byte(addr)
	// Flip the last hex char so the checksum mismatches.
	if tampered[len(tampered)-1] == '0' {
		tampered[len(tampered)-1] = '1'
	} else {
		tampered[len(tampered)-1] = '0'
	}
	if err := ValidateAddress(string(tampered)); err == nil {
		t.Fatal("checksum should catch a single-character typo")
	}
}

func TestDeriveAddress_Deterministic(t *testing.T) {
	w, _ := New()
	pub := w.PublicKeyBytes()
	if DeriveAddress(pub) != DeriveAddress(pub) {
		t.Fatal("DeriveAddress must be deterministic")
	}
}

func TestSaveLoad_Roundtrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "w.json")

	w1, _ := New()
	if err := w1.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}

	w2, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if w1.Address() != w2.Address() {
		t.Fatalf("loaded wallet address %s != original %s", w2.Address(), w1.Address())
	}

	hash := bytes32()
	sig, _ := w2.Sign(hash)
	ok, _ := Verify(w1.PublicKeyHex(), hex.EncodeToString(hash), sig)
	if !ok {
		t.Fatal("signature from loaded wallet doesn't verify against original pubkey")
	}
}

func TestSave_FilePermissions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "w.json")
	w, _ := New()
	if err := w.Save(path); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("wallet file perm = %o, want 0600", info.Mode().Perm())
	}
}

func TestLoadOrCreate_NewFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "new.json")
	w, created, err := LoadOrCreate(path)
	if err != nil {
		t.Fatal(err)
	}
	if !created {
		t.Fatal("expected created=true for a missing file")
	}
	if w == nil {
		t.Fatal("wallet is nil")
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("file should exist on disk: %v", err)
	}
}

func TestLoadOrCreate_ExistingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "exists.json")

	w1, _, err := LoadOrCreate(path)
	if err != nil {
		t.Fatal(err)
	}
	addr1 := w1.Address()

	w2, created, err := LoadOrCreate(path)
	if err != nil {
		t.Fatal(err)
	}
	if created {
		t.Fatal("expected created=false when the file already exists")
	}
	if w2.Address() != addr1 {
		t.Fatalf("LoadOrCreate returned different wallet on second call: %s vs %s", w2.Address(), addr1)
	}
}

func TestLoad_InvalidFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(path, []byte("not json"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("expected Load to fail on invalid JSON")
	}
}

func bytes32() []byte {
	b := make([]byte, 32)
	for i := range b {
		b[i] = byte(i + 1)
	}
	return b
}
