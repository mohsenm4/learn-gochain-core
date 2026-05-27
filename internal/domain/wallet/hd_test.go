package wallet

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestMnemonic_RoundTrip(t *testing.T) {
	entropy := make([]byte, EntropyBytes)
	for i := range entropy {
		entropy[i] = byte(i * 7)
	}
	m, err := EncodeMnemonic(entropy)
	if err != nil {
		t.Fatalf("EncodeMnemonic: %v", err)
	}
	if got := len(strings.Fields(m)); got != MnemonicWords {
		t.Fatalf("expected %d words, got %d (%q)", MnemonicWords, got, m)
	}
	back, err := DecodeMnemonic(m)
	if err != nil {
		t.Fatalf("DecodeMnemonic: %v", err)
	}
	if string(back) != string(entropy) {
		t.Fatalf("entropy round-trip mismatch")
	}
}

func TestMnemonic_DetectsTypo(t *testing.T) {
	m, err := NewMnemonic()
	if err != nil {
		t.Fatal(err)
	}
	words := strings.Fields(m)
	// Swap any two adjacent words — the checksum should catch it.
	words[0], words[1] = words[1], words[0]
	if _, err := DecodeMnemonic(strings.Join(words, " ")); err == nil {
		t.Fatal("checksum should reject a swapped mnemonic")
	}
}

func TestMnemonic_RejectsUnknownWord(t *testing.T) {
	m, err := NewMnemonic()
	if err != nil {
		t.Fatal(err)
	}
	words := strings.Fields(m)
	words[0] = "notaword"
	if _, err := DecodeMnemonic(strings.Join(words, " ")); err == nil {
		t.Fatal("decoding should reject unknown words")
	}
}

func TestHD_DeterministicFromMnemonic(t *testing.T) {
	h1, mnemonic, err := NewHDWallet()
	if err != nil {
		t.Fatal(err)
	}
	h2, err := HDFromMnemonic(mnemonic, "")
	if err != nil {
		t.Fatal(err)
	}
	for i := uint32(0); i < 5; i++ {
		w1, err := h1.DeriveReceive(i)
		if err != nil {
			t.Fatal(err)
		}
		w2, err := h2.DeriveReceive(i)
		if err != nil {
			t.Fatal(err)
		}
		if w1.Address() != w2.Address() {
			t.Fatalf("receive %d: same mnemonic produced different addresses (%s vs %s)", i, w1.Address(), w2.Address())
		}
	}
}

func TestHD_PassphraseSeparatesWallets(t *testing.T) {
	_, mnemonic, err := NewHDWallet()
	if err != nil {
		t.Fatal(err)
	}
	noPass, err := HDFromMnemonic(mnemonic, "")
	if err != nil {
		t.Fatal(err)
	}
	withPass, err := HDFromMnemonic(mnemonic, "secret")
	if err != nil {
		t.Fatal(err)
	}
	a1, _ := noPass.DeriveReceive(0)
	a2, _ := withPass.DeriveReceive(0)
	if a1.Address() == a2.Address() {
		t.Fatal("different passphrases must derive different addresses")
	}
}

func TestHD_ReceiveAndChangeBranchesDiffer(t *testing.T) {
	h, _, err := NewHDWallet()
	if err != nil {
		t.Fatal(err)
	}
	r, _ := h.DeriveReceive(0)
	c, _ := h.DeriveChange(0)
	if r.Address() == c.Address() {
		t.Fatal("receive and change branches must produce different addresses at the same index")
	}
}

func TestHD_NextReceiveAdvancesCounter(t *testing.T) {
	h, _, err := NewHDWallet()
	if err != nil {
		t.Fatal(err)
	}
	first, err := h.NextReceive()
	if err != nil {
		t.Fatal(err)
	}
	second, err := h.NextReceive()
	if err != nil {
		t.Fatal(err)
	}
	if first.Address() == second.Address() {
		t.Fatal("NextReceive must hand out a different address each call")
	}
	// Re-derive should match the first one.
	zero, err := h.DeriveReceive(0)
	if err != nil {
		t.Fatal(err)
	}
	if zero.Address() != first.Address() {
		t.Fatalf("DeriveReceive(0) should match the first NextReceive (%s vs %s)", zero.Address(), first.Address())
	}
}

func TestHD_XPubMatchesXPriv(t *testing.T) {
	h, _, err := NewHDWallet()
	if err != nil {
		t.Fatal(err)
	}
	xpub := h.XPub()
	for i := uint32(0); i < 5; i++ {
		w, err := h.DeriveReceive(i)
		if err != nil {
			t.Fatal(err)
		}
		addr, err := xpub.DeriveReceive(i)
		if err != nil {
			t.Fatal(err)
		}
		if w.Address() != addr {
			t.Fatalf("receive %d: xpub-derived address %s != xprv-derived %s", i, addr, w.Address())
		}
	}
	// And the same for the change branch.
	w, _ := h.DeriveChange(3)
	addr, _ := xpub.DeriveChange(3)
	if w.Address() != addr {
		t.Fatalf("change branch: xpub %s != xprv %s", addr, w.Address())
	}
}

func TestHD_XPubEncodeDecode(t *testing.T) {
	h, _, err := NewHDWallet()
	if err != nil {
		t.Fatal(err)
	}
	encoded := h.XPub().Encode()
	if !strings.HasPrefix(encoded, "xpub") {
		t.Fatalf("encoded xpub should start with 'xpub', got %q", encoded)
	}
	decoded, err := DecodeXPub(encoded)
	if err != nil {
		t.Fatalf("DecodeXPub: %v", err)
	}
	want, _ := h.XPub().DeriveReceive(2)
	got, _ := decoded.DeriveReceive(2)
	if want != got {
		t.Fatalf("decoded xpub derived different address: %s vs %s", got, want)
	}
}

func TestHD_SaveLoadRestoresIndices(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hd.json")

	h, mnemonic, err := NewHDWallet()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := h.NextReceive(); err != nil {
		t.Fatal(err)
	}
	if _, err := h.NextReceive(); err != nil {
		t.Fatal(err)
	}
	if _, err := h.NextChange(); err != nil {
		t.Fatal(err)
	}
	if err := h.Save(path); err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadHDWallet(path, "")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Mnemonic() != mnemonic {
		t.Fatalf("mnemonic mismatch after load")
	}
	next, err := loaded.NextReceive()
	if err != nil {
		t.Fatal(err)
	}
	expected, err := h.DeriveReceive(2)
	if err != nil {
		t.Fatal(err)
	}
	if next.Address() != expected.Address() {
		t.Fatalf("NextReceive after load should resume at index 2: got %s want %s", next.Address(), expected.Address())
	}
}

func TestHD_DerivedWalletCanSign(t *testing.T) {
	h, _, err := NewHDWallet()
	if err != nil {
		t.Fatal(err)
	}
	w, err := h.NextReceive()
	if err != nil {
		t.Fatal(err)
	}
	hash := bytes32()
	sig, err := w.Sign(hash)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	ok, err := Verify(w.PublicKeyHex(), encodeHex(hash), sig)
	if err != nil || !ok {
		t.Fatalf("derived wallet signature did not verify: ok=%v err=%v", ok, err)
	}
}

func encodeHex(b []byte) string {
	const hexChars = "0123456789abcdef"
	out := make([]byte, len(b)*2)
	for i, v := range b {
		out[i*2] = hexChars[v>>4]
		out[i*2+1] = hexChars[v&0x0f]
	}
	return string(out)
}
