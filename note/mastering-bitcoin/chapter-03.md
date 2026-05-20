# 📙 Chapter 3 — Bitcoin Core: The Reference Implementation

## The Big Picture

Bitcoin Core is the **reference software**. Other clients copy its behavior.
If Bitcoin Core does X, "real Bitcoin" does X.

Satoshi wrote the first version. Now the community keeps it.
It is a **full verification node**: it checks every rule, every block, every transaction.

---

## What's Inside Bitcoin Core?

Bitcoin Core is **many programs in one box**:

| Part                   | Job                                                  |
| ---------------------- | ---------------------------------------------------- |
| Peer Discovery         | Find other nodes on the network                      |
| Connection Manager     | Keep TCP connections to peers                        |
| Mempool                | Holds unconfirmed transactions                       |
| Validation Engine      | Checks transactions and blocks against all rules     |
| Storage Engine         | Saves blocks, headers, UTXO set (called "Coins")     |
| Miner                  | Builds candidate blocks                              |
| Wallet                 | Keys, addresses, balances                            |
| RPC interface          | JSON-RPC for apps and `bitcoin-cli`                  |

Like the **engine room** of Bitcoin — every piece does one clear job.

---

## BIPs — Bitcoin Improvement Proposals

Ideas for changes are written as **BIPs**. Like RFCs for the internet.

- Anyone can write one.
- BIP9 = a way to roll out big changes safely.

---

## Building Bitcoin Core From Source

The book walks through the build steps:

```bash
git clone https://github.com/bitcoin/bitcoin.git
cd bitcoin
git checkout v24.0.1     # pick a stable tag
./autogen.sh
./configure              # flags: --prefix=$HOME, --disable-wallet, --with-gui=no
make
make check && sudo make install
```

You get three programs:

- `bitcoind` — the daemon (server)
- `bitcoin-cli` — the command-line client (talks to bitcoind over JSON-RPC)
- `bitcoin-tx` — a helper for building raw transactions

---

## Running a Node

- ~500 GB disk for the full chain (in 2023).
- ~400 MB/day bandwidth.
- Even a Raspberry Pi can run one.

**Why run your own node?**

1. Don't trust strangers — verify yourself.
2. Privacy: nobody knows which transactions you care about.
3. Build apps on your own API.
4. Make the network stronger.

---

## `bitcoin.conf` — The Configuration File

Important options:

| Option         | Meaning                                                         |
| -------------- | --------------------------------------------------------------- |
| `datadir`      | Where to store data                                             |
| `prune`        | Delete old blocks to save disk                                  |
| `txindex=1`    | Full index of every transaction (needed for `getrawtransaction`)|
| `dbcache`      | UTXO cache size (default 450 MB)                                |
| `blocksonly=1` | Only download blocks, skip relayed txs (low bandwidth)          |
| `maxmempool`   | Max mempool size                                                |
| `alertnotify`  | Run a script when an alert fires                                |

Two preset configs in the book:

- **Full-index node** — for developers and APIs.
- **Resource-constrained node** — small servers, with `prune` + `blocksonly`.

Run modes:

```bash
bitcoind -printtoconsole    # foreground
bitcoind -daemon            # background
```

---

## The Bitcoin Core API (JSON-RPC)

Everything Bitcoin Core knows, it tells you over **JSON-RPC**.
`bitcoin-cli` is the easy wrapper.

Common commands:

```bash
bitcoin-cli help                       # list all commands
bitcoin-cli getblockchaininfo          # chain state
bitcoin-cli getnetworkinfo             # peers, version
bitcoin-cli getblockhash 1000          # hash of block #1000
bitcoin-cli getrawtransaction <txid>   # raw tx data
```

Output is JSON — humans can read it, programs can parse it.

Fields from `getblockchaininfo` (the key ones):
`chain`, `blocks`, `headers`, `bestblockhash`, `difficulty`,
`verificationprogress`, `initialblockdownload`, `size_on_disk`.

---

## Alternative Clients

Bitcoin Core is C++, but Bitcoin has libraries/clients in:
**C/C++, JavaScript, Java, Python, Go, Rust, Scala, C#.**

This diversity keeps the ecosystem from depending on one codebase.

---

## What This Chapter Teaches Us

It's the **bridge** between theory and hands-on Bitcoin:

- Bitcoin Core = the reference.
- A node = build it, run it, configure it, query it.
- From here, the rest of the book uses `bitcoind` + `bitcoin-cli` as the lab.

---

## Mapping to Our Go Node (what we will mirror)

| Bitcoin Core piece     | Our project equivalent                              |
| ---------------------- | --------------------------------------------------- |
| `bitcoind`             | `cmd/node` (the daemon)                             |
| `bitcoin-cli`          | A new CLI we will add: `cmd/cli`                    |
| JSON-RPC interface     | Our HTTP/JSON API in `internal/api/http`            |
| `getblockchaininfo`    | A new `/info` endpoint                              |
| `getnetworkinfo`       | A new `/network` endpoint                           |
| `bitcoin.conf`         | Our `app.env` + `config/config.go`                  |

**Next steps in this repo:**

1. Add an `/info` endpoint that returns chain height, best hash, difficulty, mempool size, peers.
2. Add a `/network` endpoint with peer list and node id.
3. Add `cmd/cli` — a small client (`gochain-cli`) that calls these endpoints, like `bitcoin-cli`.
