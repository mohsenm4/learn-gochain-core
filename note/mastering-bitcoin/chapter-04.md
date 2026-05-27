# 📙 Chapter 4 — Keys and Addresses

## The Big Picture

Alice wants to pay Bob, but **thousands of strangers** (the full nodes) will see
the transaction. We don't want those strangers to know who Alice or Bob are.
And only Bob should be able to spend what he just received.

The trick is **public key cryptography**:

- A **private key** (a secret number) can sign.
- A **public key** (derived from the private key) can verify.
- Math runs only one way: `private → public` is easy, `public → private` is impossible.

A Bitcoin **address** is a short, human-friendly form of a public key
(or a commitment to one), so Alice doesn't have to copy a 130-character key.

---

## Private Keys

A private key is just a **random 256-bit number** — that's it.

- Range: `1` to `n - 1`, where `n ≈ 1.158 × 10⁷⁷` (a bit less than `2²⁵⁶`).
- `2²⁵⁶ ≈ 10⁷⁷` — about the number of atoms in the visible universe.
- If you lose it, you lose your bitcoins **forever**.
- If someone else gets it, **they** own your bitcoins.

🦂 **Never write your own random generator.** Always use a CSPRNG (cryptographically
secure random number generator). A weak random source = a stolen wallet.

Example hex private key:

```
1E99423A4ED27608A15A2616A2B0E9E52CED330AC530EDCC32C8FFC6A526AEDD
```

---

## Elliptic Curve Cryptography (ECC)

Bitcoin's "trap-door" function is **elliptic curve multiplication**.

Bitcoin uses a specific curve called **`secp256k1`**:

```
y² = x³ + 7   (over a finite field of prime order p)
```

Two operations on the curve:

| Operation         | What it does                                                      |
| ----------------- | ----------------------------------------------------------------- |
| Point addition    | `P₁ + P₂ = P₃` — a third point also on the curve.                 |
| Point doubling    | A line tangent at `P` intersects the curve at one other point.    |
| Scalar mult `k·G` | `G + G + … + G` (k times) — the **trap door**.                    |

The magic: `k·G` is easy to compute, but recovering `k` from the result
is computationally impossible. That's the **discrete logarithm problem**.

---

## Public Keys

The public key is just the private key multiplied by a fixed point on the curve:

```
K = k × G
```

- `k` = private key (256-bit number)
- `G` = generator point (a constant, the same for every Bitcoin user)
- `K` = a point on the curve, written as `(x, y)`

You can share `K` with the world. Nobody can reverse it to find `k`.

🦖 Many Bitcoin implementations use the **`libsecp256k1`** library for this math.

---

## From Public Key → Bitcoin Address

A raw public key is 65 bytes (130 hex chars) — too long to read over the phone.
So Bitcoin **hashes** it down to 20 bytes:

```
A = RIPEMD160( SHA256( K ) )
```

This 20-byte result is called a **public key hash** (or HASH160). It's a
**commitment** to the public key: knowing `A` proves nothing about `K`,
but anyone who later sees `K` can check it matches `A`.

> Why two hash functions? SHA256 first to be safe, then RIPEMD160 to shrink
> the output to 20 bytes. (Satoshi never explained why exactly — just rolled with it.)

---

## Output and Input Scripts (very short version)

When Alice pays Bob, her transaction creates an **output script** that says
*"only Bob's signature can spend this"*. Bob later spends it with an **input
script** that provides the signature. Two main legacy patterns:

### P2PK — Pay to Public Key (the oldest form)

```
output: <Bob's pubkey> OP_CHECKSIG
input:  <Bob's signature>
```

Simple, but the full pubkey is huge (65 bytes). Almost never used today.

### P2PKH — Pay to Public Key Hash (the classic Bitcoin address)

```
output: OP_DUP OP_HASH160 <Bob's commitment> OP_EQUALVERIFY OP_CHECKSIG
input:  <Bob's signature> <Bob's pubkey>
```

The output only stores 20 bytes (the HASH160), saving space.
Bob reveals the full pubkey only when spending.

---

## Base58check Encoding

20 raw bytes look like `f54a5851e9372b87810a8e60cdd2e7cfd80b6e31`.
Hard to read, easy to mistype. Bitcoin uses **base58check** to make it friendlier:

- **base58** alphabet = `[a-zA-Z0-9]` minus `0`, `O`, `l`, `I` (look-alikes removed).
- **checksum** = first 4 bytes of `SHA256(SHA256(prefix‖data))` — catches typos.
- **version prefix** = a single byte that tells decoders what kind of data this is.

```
[ version ] [ payload ] [ checksum ]  →  base58 encode  →  human-friendly string
```

Table of common version prefixes:

| Type                  | Prefix (hex) | First char(s) |
| --------------------- | ------------ | ------------- |
| Mainnet P2PKH address | `0x00`       | `1`           |
| Mainnet P2SH address  | `0x05`       | `3`           |
| Testnet P2PKH         | `0x6F`       | `m` or `n`    |
| Testnet P2SH          | `0xC4`       | `2`           |
| Private key (WIF)     | `0x80`       | `5`, `K`, `L` |
| BIP32 extended pubkey | `0x0488B21E` | `xpub`        |

So a mainnet address always starts with `1`, a P2SH with `3`, an extended
pubkey with `xpub`. The prefix is a quick visual cue.

---

## Compressed Public Keys

A point `(x, y)` on the curve takes **65 bytes** uncompressed (1 prefix + 32x + 32y).
But because `y² = x³ + 7`, once you know `x` there are only **two possible** `y` values.

So instead of storing `y`, store only:

- `02` prefix if `y` is **even**
- `03` prefix if `y` is **odd**

Result: a **33-byte** public key. ~50% smaller transactions. Win.

⚠️ Same private key, compressed vs uncompressed = **different addresses**.
That's why WIF-compressed format has a trailing `0x01` byte — to tell the
wallet "derive a compressed pubkey from this key."

---

## P2SH — Pay to Script Hash

Sometimes the *condition* for spending is complex (e.g. "2 of 3 signatures").
The condition itself could be huge. So instead of putting the full script in
the output, we hash it.

```
output: OP_HASH160 <script commitment> OP_EQUAL
```

When Bob spends, he reveals the **redeem script** plus whatever data it requires
(signatures, etc.). Full nodes hash the redeem script and check it matches.

- P2SH addresses start with `3` on mainnet.
- The 2012 BIP16 upgrade introduced this.
- Most common use: **multisig**, but really any script.

🦂 **Collision attacks**: 160-bit hashes only give ~80 bits of collision resistance.
For scripts where multiple parties contribute, an attacker could craft a colliding
script and steal funds. Newer address types use 256-bit hashes for this reason.

---

## Bech32 Addresses (Segwit)

In 2017, Segregated Witness (**segwit**) upgraded the protocol. To use it
properly, wallets needed a **new address format**. Bitcoin developers seized
the chance to fix several base58check problems:

| Base58check problem                                | Bech32 fix                            |
| -------------------------------------------------- | ------------------------------------- |
| Mixed case — hard to read aloud, hard to QR-encode | All lowercase                         |
| Detects errors but can't tell *where* the typo is  | Detects **and locates** errors        |
| Mixed case wastes QR code space                    | Lowercase fits a denser QR mode       |

Bech32 anatomy:

```
bc1qw508d6qejxtdg4y5r3zarvary0c5xw7kv8f3t4
└┬┘ │ └──────────────── data part + 6-char checksum ───────────────┘
 │  └─ separator (always "1")
 └─ HRP (human readable part: "bc" mainnet, "tb" testnet)
```

The **data part** has three pieces:

1. **Witness version** — 1 byte (`0` = segwit v0, `1` = taproot, …)
2. **Witness program** — the actual hash/pubkey (20 or 32 bytes typically)
3. **Checksum** — 6 chars, BCH error-correcting code (used only for *detection* in Bitcoin)

### Bech32 vs Bech32m

Bech32 had a bug: in some cases you could add/remove a `q` near the end of an
address and the checksum still passed. Fixed by changing one constant in the
algorithm — the new version is called **bech32m**.

| Use case            | Encoding |
| ------------------- | -------- |
| Segwit v0 (P2WPKH, P2WSH) | bech32   |
| Segwit v1+ (Taproot/P2TR) | bech32m  |

### Address types you'll see

| Type    | What it points to                 | Output script                              |
| ------- | --------------------------------- | ------------------------------------------ |
| P2WPKH  | 20-byte HASH160 of pubkey         | `OP_0 <20-byte-hash>`                      |
| P2WSH   | 32-byte SHA256 of script          | `OP_0 <32-byte-hash>`                      |
| P2TR    | 32-byte taproot output (curve pt) | `OP_1 <32-byte-point>`                     |

---

## Private Key Formats

The private key is always the same 256-bit number — just shown in different clothes:

| Format         | Prefix    | Description                                       |
| -------------- | --------- | ------------------------------------------------- |
| Hex            | none      | 64 hex chars                                      |
| WIF            | `5`       | base58check of `0x80 ‖ key`                       |
| WIF-compressed | `K` or `L`| base58check of `0x80 ‖ key ‖ 0x01`                |

Same number, three masks. WIF (Wallet Import Format) is what wallets use to
import/export single keys or generate QR codes.

---

## Advanced Topics

### Vanity Addresses

A Bitcoin address that contains a readable prefix, like `1LoveBPzzD…` or
`1KidsCharity…`. Generated by brute force: try random keys, check the address,
repeat billions of times.

Difficulty grows by **58×** per extra character (because base58):

| Prefix length | Average search time on a desktop |
| ------------- | -------------------------------- |
| `1K`          | < 1 ms                           |
| `1Kid`        | < 2 s                            |
| `1Kids`       | 1 minute                         |
| `1KidsCh`     | 2 days                           |
| `1KidsCharity`| 2.5 million years                |

Vanity addresses are no more or less secure than normal ones — same crypto.
But they've largely died out because **deterministic (HD) wallets** can't
import individual keys, and reusing an address hurts privacy.

### Paper Wallets

🦂 **Don't use them.** Paper wallets (a private key printed on paper) sound
romantic but have killed a lot of bitcoins:

- Random number generators on websites can have **back doors**.
- Once you spend even part of the balance, you must move *everything* off
  the paper (a UTXO quirk).
- Hard to back up safely without making them less secure.

Use a **hardware signing device** and a **BIP39 recovery seed** instead.

---

## What This Chapter Teaches Us

This chapter is the **cryptography spine** of Bitcoin:

1. **Private key** = a secret 256-bit number.
2. **Public key** = `private × G` on `secp256k1`. One-way.
3. **Address** = a short, error-checked commitment to the public key.

The address formats evolved over time:

```
P2PK  →  P2PKH  →  P2SH  →  P2WPKH/P2WSH (bech32)  →  P2TR (bech32m)
 raw      hash      script     segwit                    taproot
```

Each step made addresses **shorter, safer, or more expressive**.

---

## Mapping to Our Go Node

Our project's wallet ([internal/domain/wallet/wallet.go](../../internal/domain/wallet/wallet.go))
takes some shortcuts compared to Bitcoin. Here is the honest comparison:

| Concept             | Bitcoin                                | Our Go node                                        |
| ------------------- | -------------------------------------- | -------------------------------------------------- |
| Curve               | `secp256k1`                            | `P-256` (Go stdlib)                                |
| Signature scheme    | ECDSA + Schnorr (taproot)              | ECDSA only                                         |
| Private key size    | 256 bits                               | 256 bits                                           |
| Public key encoding | 33 bytes compressed (`02`/`03` prefix) | 33 bytes compressed (`elliptic.MarshalCompressed`) |
| Address derivation  | `RIPEMD160(SHA256(pubkey))`            | `RIPEMD160(SHA256(pubkey))` — same as Bitcoin      |
| Address format      | base58check or bech32/bech32m          | hex string with `0x` prefix                        |
| Error detection     | base58check / bech32 checksum          | 4-byte `SHA256(SHA256(payload))[:4]` checksum      |

**Things we could add later if we want to look more "real":**

1. Switch to `secp256k1` (drop `crypto/elliptic`, pull in a `secp256k1` lib).
2. Replace the hex `0x…` address with **base58check** or **bech32**. The 4-byte
   checksum is already in place — base58 would just be a different encoding of
   the same payload.
3. Add a **WIF**-like format for exporting/importing single keys (we currently
   store the raw hex private key in the wallet JSON).

The chapter gives us the recipe; the next chapters (HD wallets, seeds, BIP39)
will tell us how real wallets do this at scale.
