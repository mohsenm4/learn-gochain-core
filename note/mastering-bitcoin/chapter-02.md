# 📗 Chapter 2 — How Bitcoin Works: A Real Example

## The Main Idea

Chapter 2 follows **one real payment** from start to end. We watch the money move through the network.

---

## The New Story (Alice buys something)

1. Alice wants to buy a podcast episode from **Bob's online store**.
2. Bob accepts bitcoin. The website shows a price in **dollars AND in BTC**.
3. The site makes a **QR code** with an invoice (address + amount + label).
4. Alice scans it with her phone. She presses **Send**.
5. A few seconds later, Bob sees the payment.
6. Then we follow what happens inside Bitcoin during those seconds.

---

## What is a Transaction?

A transaction has 2 sides, like a **double-entry bookkeeping**:

- **Inputs** = money coming IN (from old payments).
- **Outputs** = money going OUT (to new owners).

**Important:** Inputs total is a little **BIGGER** than outputs total. The difference is the **transaction fee** for the miner.

Example:

```text
Inputs:  0.55 BTC
Outputs: 0.50 BTC
Fee:     0.05 BTC  → goes to miner
```

## Transaction Chains

Bitcoin is a **chain of transactions**:

```text
Joe → Alice → Bob → Gopesh (Bob's web designer)
 Tx1    Tx2    Tx3
```

The **output** of one transaction becomes the **input** of the next. Money flows like a chain.

## Change Output (پول خرد)

Just like with cash:

- You have a 20$ bill.
- You buy a 5$ item.
- You get 15$ change.

Same in Bitcoin: if Alice has 100,000 sats but only wants to pay 75,000, the wallet sends 20,000 sats back to a **change address** of Alice. (5,000 is the fee.)

## UTXO (very important word)

**UTXO = Unspent Transaction Output.**

Your "balance" is not one number. It is a **collection of unspent outputs** the wallet adds together. Like coins in your pocket.

---

## Building a Transaction — 4 Steps Alice's Wallet Does

1. **Get the right inputs** → find UTXOs she owns.
2. **Create the outputs** → one for Bob, one back to Alice (change).
3. **Sign** it with Alice's private key (proof of ownership).
4. **Send** it to the Bitcoin network.

The wallet can build the transaction **offline**. Like writing a check at home and mailing it later.

## How the Transaction Travels — "Gossiping"

- Alice's wallet sends the transaction to **one** Bitcoin node.
- That node sends it to many other nodes.
- They send to more nodes.
- In a few seconds, most computers in the world have heard about it.
- This spreading is called **gossiping**.

---

## Mining — How It Becomes "Confirmed"

**Mining = a decentralized lottery.**

1. Miners collect new transactions.
2. They put them in a **candidate block**.
3. They try to find a special number using a **hash function** (a math puzzle).
4. The chance of winning is tiny — they must try about **168 billion trillion** times.
5. The winner shouts: "I found it!"
6. Everyone checks the answer (easy — only 1 calculation).
7. The block joins the **blockchain**.

This is called **Proof of Work (PoW)** — hard to make, easy to check.

### Why does this protect Bitcoin?

To cheat (change a past payment), someone must redo **ALL** the work for that block AND every block after. This is too expensive. So the chain is safe.

## Confirmations

- Block #1 with Alice's transaction = **1 confirmation**
- Next block built on top = **2 confirmations**
- After **6 confirmations** → very hard to change. Safe.

The first block ever (#0) is called the **genesis block**.

---

## Bob Spends the Money

- The blockchain now has Alice's payment.
- Bob now owns that output.
- Bob pays his web designer **Gopesh** using this same money.

The chain continues: **Joe → Alice → Bob → Gopesh.**
