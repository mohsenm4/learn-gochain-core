# 📔 Chapter 5 — Wallet Recovery

## The Big Picture (در یک خط)

A wallet **does not hold coins** — it holds **keys**. Lose the keys, lose the
coins. This chapter is about how a single 12-word phrase can stand in for
millions of keys, and how to back that phrase up safely.

---

## The Core Problem

If your wallet database is lost or corrupted, the bitcoins it controlled
become unspendable. No bank, no support team, no recovery service can bring
them back. So we need a way to **back up the keys** that is:

- Small enough to write on paper
- Strong enough that nobody can guess it
- Reusable on any compatible wallet

This entire chapter is about that one problem.

---

## Two Ways to Generate Keys

### The old way — Independent (nondeterministic) key generation

Every time the wallet needs a new address, it picks a fresh random key.

🦂 Why this is painful:

- Each new key means a fresh backup — miss one and the money received on that
  key is gone forever.
- ~32 bytes of data per key, plus overhead.
- Encourages address reuse (which hurts privacy) just to avoid more backups.

```text
Wallet:  [k1] [k2] [k3] [k4] [k5] ...
         all random, all unrelated, all must be backed up
```

### The modern way — Deterministic key generation

Take one random number (a **seed**), and derive every key from it using a
hash function.

```text
seed  →  k1, k2, k3, k4, ...  (always the same keys for the same seed)
```

Because a hash function always produces the same output for the same input,
the seed alone is enough to recreate the entire wallet later.

✅ **Back up once, recover forever.** Just save the seed.

---

## HD Wallets — A Tree of Keys (BIP32)

Every modern Bitcoin wallet uses **Hierarchical Deterministic (HD)** wallets.
Instead of a flat list of keys, the seed grows a **tree**:

```text
              seed
               │
          master key (m)
         /     |     |     \
       m/0   m/1   m/2   m/3      ← children
       /|\   /|\
   m/0/0 m/0/1 ...                ← grandchildren
```

- Each key can have up to 4 billion children.
- The tree can be as deep as you want.
- Every key in the tree is derived from the same seed.

**Why a tree?** Different branches can serve different purposes —
one branch for receiving, one for change, one for a separate business
account, one per cryptocurrency.

---

## Recovery Codes — 12 Words for a Wallet (BIP39)

A seed is a huge random number, e.g.:

```text
0C1E24E59177D297E14D45F14E1A1A
```

Nobody can memorize that or copy it without errors. So BIP39 turns it into
**12 (or 24) simple words**:

```text
army van defense carry jealous true
garbage claim echo media make crunch
```

These 12 words **are** the seed (re-encoded). Anyone with the words can
restore the entire wallet.

### How the words are generated (BIP39)

Six steps:

1. Pick 128–256 random bits (entropy).
2. Compute a small checksum (first N bits of `SHA256(entropy)`), where
   N = entropy_size / 32.
3. Append the checksum to the entropy.
4. Split the result into groups of **11 bits**.
5. Each 11-bit group is a number 0–2047. Look it up in the standard
   2048-word list.
6. The sequence of words is the recovery code.

| Entropy bits | Checksum bits | Total bits | Words |
| ------------ | ------------- | ---------- | ----- |
| 128          | 4             | 132        | 12    |
| 160          | 5             | 165        | 15    |
| 192          | 6             | 198        | 18    |
| 224          | 7             | 231        | 21    |
| 256          | 8             | 264        | 24    |

### From words back to a seed

The words themselves are not the seed used by BIP32. They are stretched
through a slow function:

```text
12 words  +  (optional passphrase)
              ↓
          PBKDF2 with HMAC-SHA512
              ↓
        512-bit seed
```

The 2,048 rounds of hashing are deliberately slow. Why? So that brute-force
attacks against a partially-known recovery code are painful, even with
specialized hardware.

---

## Optional Passphrase

You can add a passphrase on top of the 12 words. The passphrase is **not
part of the words** — it's a separate secret you must remember.

Key insight: **there is no "wrong" passphrase**. Every passphrase produces
a different valid wallet. Combined with your 12 words, each passphrase
creates a separate, independent wallet tree.

🛡️ **What this gives you:**

- If a thief finds the 12 words but not the passphrase, the bitcoins stay
  safe.
- If you're coerced into revealing your words, you can hand over a "duress"
  passphrase that leads to a near-empty wallet. The attacker has no way to
  prove there's more. (This is called **plausible deniability**.)

⚠️ **The cost:** if you forget the passphrase, the funds are gone. There is
no recovery. Most schemes (BIP39, Electrum v2, SLIP39) do not validate the
passphrase — they just produce a different empty wallet, with no warning.

---

## Other Recovery Code Schemes

BIP39 is the most popular, but not the only one.

| Scheme          | Where used            | Notable feature                                        |
| --------------- | --------------------- | ------------------------------------------------------ |
| **BIP39**       | most modern wallets   | universal, but no version field                        |
| **Electrum v2** | Electrum              | versioning baked in, no global word list               |
| **Aezeed**      | LND (Lightning)       | stores wallet birthday, validates passphrase           |
| **SLIP39**      | newer, by SatoshiLabs | seed can be split across multiple shares               |
| **Codex32**     | very new              | can be generated and verified using only pen and paper |

---

## Backing Up More Than Keys

A seed regenerates every key, but **not** the extra data wallets often store:

- Transaction labels ("paid Bob for podcast")
- Lightning Network channel state
- Custom protocol metadata

🦂 If you only back up your 12 words, you'll restore the funds but lose all
the labels and history that helped you remember what was what. Some wallets
solve this by encrypting a full backup with a key derived from the seed —
that encrypted blob can then live on any cloud service.

---

## Creating an HD Wallet from the Seed

Given the seed, how do we actually grow the tree?

### Step 1 — derive the master key

```text
seed → HMAC-SHA512 → 512-bit output
                       │
             ┌─────────┴─────────┐
       left 256 bits        right 256 bits
       = master private     = master chain code
         key (m)              (c)
```

What is the **chain code**? Extra entropy needed alongside the key to
derive its children. Without the chain code, you cannot continue the tree.

### Step 2 — Child Key Derivation (CKD)

To derive a child from a parent, mix three things through HMAC-SHA512:

```text
parent_key  +  parent_chain_code  +  index_number (32-bit)
                       ↓
                  HMAC-SHA512
                       ↓
            child_key  +  child_chain_code
```

Change the `index` to get sibling children (child 0, child 1, child 2, …).
Each parent can produce ~2 billion normal children (indices 0 to 2³¹−1).
The other half of the index range is reserved for *hardened* derivation
(see below).

---

## Extended Keys — xprv and xpub

Because a key alone is not enough to derive children (the chain code is
also required), wallets package them together as an **extended key**:

- **xprv** = private key + chain code (everything you need)
- **xpub** = public key + chain code (read-only)

They're encoded in base58check, with the recognizable `xprv` / `xpub`
prefixes:

```text
xpub67pozcx8pe95XVuZLHXZeG6XWXHpGq6QvScmNfl7cS5mtjJ2tgypeQbBs2UAR6KECeeMVKZBP…
```

---

## The Magic of xpub

The headline feature of HD wallets: from an **xpub**, you can derive an
unlimited number of **child public keys** (and addresses) — without ever
having the private key.

### Real-world example: Gabriel's web store

Gabriel runs a Bitcoin-accepting online shop. Early on, every customer paid
to the same address, which made bookkeeping a nightmare and leaked
information about all his customers.

With an HD wallet:

1. He exports the xpub from his hardware signing device.
2. He loads only the xpub onto the web server.
3. The server derives a fresh address for every new order.
4. The private keys never touch the server. If the server is hacked,
   the attacker can read incoming addresses but **cannot spend the funds**.

```text
web server (xpub):           can derive addresses ✅   can spend ❌
hardware device (xprv):      can spend ✅
```

---

## Hardened Derivation — A Firewall Between Levels

There's a subtle attack on plain (normal) derivation. If an attacker has:

- the parent **xpub**, and
- one child **private key**

…they can recover the parent **private key**. That would compromise every
sibling and grandchild too. Bad.

### The fix: hardened derivation

Use the parent's **private key** (instead of its public key) as input to
the HMAC. Now, even leaking a child private key reveals nothing about the
parent.

```text
normal:    parent_public_key  + chain_code + index → child
hardened:  parent_private_key + chain_code + index → child
```

This puts a "firewall" between the parent and its hardened children.

### Indexing convention

The 32-bit index is split into two halves:

- `0` to `2³¹−1` → normal derivation
- `2³¹` to `2³²−1` → hardened derivation

Hardened indices are written with a trailing apostrophe (`'`) for human
readability:

- `m/0` = first normal child
- `m/0'` = first hardened child (internally index `2³¹`)

**Best practice:** the first level under master is always hardened, so the
master key itself is never reachable from any descendant xpub.

---

## HD Wallet Paths

Each key in the tree has a path, read like a file path:

```text
m / purpose' / coin_type' / account' / change / address_index
```

Example: `m/44'/0'/0'/0/5` means:

- `m` — start at master
- `44'` — purpose: BIP44
- `0'` — coin: Bitcoin (1' is testnet)
- `0'` — account 0
- `0` — receive branch (1 would be the change branch)
- `5` — the 6th address (zero-indexed)

`M` (capital) means the public version — `M/0/3` is a public-key path.

### The change branch

Each account has two sub-branches at level 4:

- `0` — receive addresses (you give these to others)
- `1` — change addresses (your wallet sends leftover funds here)

Splitting them keeps bookkeeping clean and improves privacy.

### Common standards (BIP43, BIP44, etc.)

The `purpose'` level tells the wallet which script type to use:

| Standard | Script type     | Purpose path     |
| -------- | --------------- | ---------------- |
| BIP44    | P2PKH (legacy)  | `m/44'/0'/0'`    |
| BIP49    | Nested P2WPKH   | `m/49'/0'/0'`    |
| BIP84    | P2WPKH (segwit) | `m/84'/0'/0'`    |
| BIP86    | P2TR (taproot)  | `m/86'/0'/0'`    |

This is why the *same seed* in two wallets that use different defaults can
appear to have different balances — they're scanning different branches.

---

## Implicit vs. Explicit Paths

When recovering from a seed, the wallet needs to know **which paths** to
scan. Two strategies:

- **Implicit paths**: hardcoded standard paths (BIP44/49/84/86). The wallet
  always checks these. Simple, but inflexible — non-standard scripts (like
  multisig) can be missed.
- **Explicit paths**: paths are recorded alongside the recovery code, often
  using **output script descriptors** (BIPs 380–389). More flexible, more
  to back up.

A descriptor looks like:

```text
pkh([d34db33f/44'/0'/0']xpub6ERA…RcEL/1/*)
```

This says: "derive P2PKH keys from this xpub at path `1/*`, and it
originally came from fingerprint `d34db33f` at HD path `m/44'/0'/0'`."

---

## The Gap Limit

A wallet doesn't generate billions of addresses up front. It generates,
say, the first 100. When a payment arrives at address 1, it generates
address 101. The window slides forward.

But what if 21 consecutive addresses get no payments? The wallet stops
scanning. The **gap limit** (commonly 20) is the maximum number of empty
addresses in a row before the wallet gives up.

This matters during recovery: a wallet restoring from a seed will miss any
funds received past the gap. Wallets that hand out many invoices (like
e-commerce sites with BTCPay Server) raise their gap limit.

---

## Putting It All Together

```text
12 words (recovery code)
    │
    │ PBKDF2 + optional passphrase
    ▼
512-bit seed
    │
    │ HMAC-SHA512 with key "Bitcoin seed"
    ▼
master private key + master chain code
    │
    │ Child Key Derivation (repeated)
    ▼
tree of extended keys
    │
    │ BIP44 / 49 / 84 / 86 conventions
    ▼
receive addresses, change addresses, separate accounts, multi-currency
```

---

## Best-Practice Recap

1. Use an HD wallet with a BIP39 (or better) recovery code.
2. Write the words on paper or steel. Never put them in cloud, email, or photos.
3. Use a passphrase if you understand the trade-offs (loss = permanent loss).
4. Keep `xprv` offline (hardware device). Load only `xpub` on web servers.
5. Always harden the first level below master.
6. Test your backups — restore on a different device and verify the balance.

---

## Mapping to Our Go Node

| Concept               | Bitcoin (this chapter)                     | Our Go node                                |
| --------------------- | ------------------------------------------ | ------------------------------------------ |
| Seed                  | 128–512 random bits                        | none — each key is generated independently |
| Recovery code (BIP39) | 12–24 words encoding the seed              | none — keys saved as raw JSON              |
| HD key derivation     | BIP32 tree                                 | flat: one key per wallet                   |
| Child key derivation  | HMAC-SHA512(parent_key, chain_code, index) | none                                       |
| Extended keys         | xprv / xpub (base58check)                  | none                                       |
| Multi-account paths   | BIP44 `m/44'/coin'/account'/…`             | none                                       |
| Hardened derivation   | index ≥ 2³¹                                | n/a                                        |
| Address gap limit     | typically 20                               | n/a                                        |

**Things we could add later to look more "real":**

1. Generate a random seed instead of a raw key, and persist the seed.
2. Implement BIP39 — turn the seed into a 12-word recovery code.
3. Add a minimal HD derivation (single chain code, child index counter).
4. Honor the receive/change split — separate branches for each.
5. Export an xpub for a read-only "view" of the wallet.

This chapter is the answer to the question chapter 4 left hanging:
*"OK, I can generate one key safely — but how do I manage thousands of them
without drowning in backups?"* The answer is the seed, and everything that
grows out of it.
