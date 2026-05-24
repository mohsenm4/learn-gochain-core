package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/Mohsen20031203/learn-gochain-core/internal/domain/transaction"
	"github.com/Mohsen20031203/learn-gochain-core/internal/domain/wallet"
)

const defaultAPI = "http://localhost:9090"

type command struct {
	name    string
	usage   string
	summary string
	run     func(api string, args []string) error
}

var commands []command

func init() {
	commands = []command{
		{
			name:    "help",
			usage:   "help",
			summary: "list available commands",
			run:     func(api string, args []string) error { printHelp(); return nil },
		},
		{
			name:    "getinfo",
			usage:   "getinfo",
			summary: "chain state: height, best hash, difficulty, mempool size",
			run:     func(api string, args []string) error { return getJSON(api + "/info") },
		},
		{
			name:    "getnetworkinfo",
			usage:   "getnetworkinfo",
			summary: "node id, tcp address, peers",
			run:     func(api string, args []string) error { return getJSON(api + "/network") },
		},
		{
			name:    "getminerinfo",
			usage:   "getminerinfo",
			summary: "miner wallet address used for coinbase outputs",
			run:     func(api string, args []string) error { return getJSON(api + "/walletinfo") },
		},
		{
			name:    "getchain",
			usage:   "getchain",
			summary: "full chain as JSON",
			run:     func(api string, args []string) error { return getJSON(api + "/chain") },
		},
		{
			name:    "getmempool",
			usage:   "getmempool",
			summary: "transactions currently in the mempool",
			run:     func(api string, args []string) error { return getJSON(api + "/mempool") },
		},
		{
			name:    "getblock",
			usage:   "getblock <hash>",
			summary: "fetch a block by its hash",
			run: func(api string, args []string) error {
				if len(args) < 1 {
					return fmt.Errorf("getblock requires <hash>")
				}
				return getJSON(api + "/block/" + url.PathEscape(args[0]))
			},
		},
		{
			name:    "gettransaction",
			usage:   "gettransaction <txid>",
			summary: "status (confirmations, block index) of a tx",
			run: func(api string, args []string) error {
				if len(args) < 1 {
					return fmt.Errorf("gettransaction requires <txid>")
				}
				return getJSON(api + "/transactions/" + url.PathEscape(args[0]))
			},
		},
		{
			name:    "getbalance",
			usage:   "getbalance <address>",
			summary: "balance for an address",
			run: func(api string, args []string) error {
				if len(args) < 1 {
					return fmt.Errorf("getbalance requires <address>")
				}
				return getJSON(api + "/balance/" + url.PathEscape(args[0]))
			},
		},
		{
			name:    "getutxos",
			usage:   "getutxos <address>",
			summary: "unspent outputs for an address",
			run: func(api string, args []string) error {
				if len(args) < 1 {
					return fmt.Errorf("getutxos requires <address>")
				}
				return getJSON(api + "/utxos/" + url.PathEscape(args[0]))
			},
		},
		{
			name:    "mine",
			usage:   "mine",
			summary: "manually trigger mining (creates a block, even if mempool is empty)",
			run: func(api string, args []string) error {
				return postJSON(api+"/mine", nil)
			},
		},
		{
			name:    "createwallet",
			usage:   "createwallet <path>",
			summary: "create a new ECDSA keypair and save it to <path>",
			run: func(api string, args []string) error {
				if len(args) < 1 {
					return fmt.Errorf("createwallet requires <path>")
				}
				path := args[0]
				if _, err := os.Stat(path); err == nil {
					return fmt.Errorf("file already exists: %s (refusing to overwrite)", path)
				}
				w, err := wallet.New()
				if err != nil {
					return err
				}
				if err := w.Save(path); err != nil {
					return err
				}
				fmt.Printf("wallet created\n  path:    %s\n  address: %s\n", path, w.Address())
				return nil
			},
		},
		{
			name:    "getaddress",
			usage:   "getaddress <wallet_path>",
			summary: "print the address of a saved wallet",
			run: func(api string, args []string) error {
				if len(args) < 1 {
					return fmt.Errorf("getaddress requires <wallet_path>")
				}
				w, err := wallet.Load(args[0])
				if err != nil {
					return err
				}
				fmt.Println(w.Address())
				return nil
			},
		},
		{
			name:    "send",
			usage:   "send <from_wallet> <to_address> <amount>",
			summary: "build, sign, and submit a transaction (optional 4th arg = fee, default 0)",
			run:     runSend,
		},
	}
}

func runSend(api string, args []string) error {
	if len(args) < 3 {
		return fmt.Errorf("send requires <from_wallet> <to_address> <amount> [fee]")
	}
	walletPath := args[0]
	toAddr := args[1]
	amount, err := strconv.ParseUint(args[2], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid amount: %w", err)
	}
	var fee uint64
	if len(args) >= 4 {
		fee, err = strconv.ParseUint(args[3], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid fee: %w", err)
		}
	}

	w, err := wallet.Load(walletPath)
	if err != nil {
		return fmt.Errorf("load wallet: %w", err)
	}
	from := w.Address()

	// Fetch UTXOs owned by the sender.
	var utxoResp struct {
		Address string `json:"address"`
		UTXOs   []struct {
			Key struct {
				TxID  string `json:"TxID"`
				Index int    `json:"Index"`
			} `json:"Key"`
			Output struct {
				Value   uint64 `json:"value"`
				Address string `json:"address"`
			} `json:"Output"`
		} `json:"utxos"`
	}
	if err := getJSONInto(api+"/utxos/"+url.PathEscape(from), &utxoResp); err != nil {
		return fmt.Errorf("fetch utxos: %w", err)
	}

	need := amount + fee
	var inputs []transaction.TxInput
	var total uint64
	for _, u := range utxoResp.UTXOs {
		inputs = append(inputs, transaction.TxInput{
			TxID:     u.Key.TxID,
			OutIndex: u.Key.Index,
		})
		total += u.Output.Value
		if total >= need {
			break
		}
	}
	if total < need {
		return fmt.Errorf("insufficient funds: have %d, need %d (amount=%d fee=%d)", total, need, amount, fee)
	}

	outputs := []transaction.TxOutput{
		{Value: amount, Address: toAddr},
	}
	if change := total - need; change > 0 {
		outputs = append(outputs, transaction.TxOutput{Value: change, Address: from})
	}

	tx := transaction.NewTransaction(inputs, outputs)

	sigHex, err := w.Sign(tx.SigningHash())
	if err != nil {
		return fmt.Errorf("sign: %w", err)
	}
	pubHex := w.PublicKeyHex()
	for i := range tx.Inputs {
		tx.Inputs[i].Signature = sigHex
		tx.Inputs[i].PubKey = pubHex
	}
	tx.ID = tx.ComputeID()

	body, _ := json.Marshal([]*transaction.Transaction{tx})
	resp, err := http.Post(api+"/transactions", "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("post: %w", err)
	}
	defer resp.Body.Close()
	out, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return fmt.Errorf("%s: %s", resp.Status, strings.TrimSpace(string(out)))
	}
	fmt.Printf("submitted tx %s\nfrom:    %s\nto:      %s\namount:  %d\nfee:     %d\nchange:  %d\nresponse: %s\n",
		tx.ID, from, toAddr, amount, fee, total-need, strings.TrimSpace(string(out)))
	return nil
}

func main() {
	fs := flag.NewFlagSet("gochain-cli", flag.ExitOnError)
	api := fs.String("api", envOr("GOCHAIN_API", defaultAPI), "node HTTP API base URL")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: gochain-cli [--api URL] <command> [args]\n\n")
		printHelp()
	}
	_ = fs.Parse(os.Args[1:])

	args := fs.Args()
	if len(args) == 0 {
		fs.Usage()
		os.Exit(1)
	}

	name := args[0]
	for _, cmd := range commands {
		if cmd.name == name {
			if err := cmd.run(strings.TrimRight(*api, "/"), args[1:]); err != nil {
				fmt.Fprintln(os.Stderr, "error:", err)
				os.Exit(1)
			}
			return
		}
	}

	fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", name)
	printHelp()
	os.Exit(1)
}

func printHelp() {
	fmt.Fprintln(os.Stderr, "commands:")
	for _, cmd := range commands {
		fmt.Fprintf(os.Stderr, "  %-44s %s\n", cmd.usage, cmd.summary)
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getJSON(endpoint string) error {
	resp, err := http.Get(endpoint)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("%s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	var pretty any
	if json.Unmarshal(body, &pretty) == nil {
		out, _ := json.MarshalIndent(pretty, "", "  ")
		fmt.Println(string(out))
		return nil
	}
	fmt.Println(string(body))
	return nil
}

func getJSONInto(endpoint string, v any) error {
	resp, err := http.Get(endpoint)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("%s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	return json.Unmarshal(body, v)
}

func postJSON(endpoint string, body []byte) error {
	var r io.Reader
	if body != nil {
		r = bytes.NewReader(body)
	}
	resp, err := http.Post(endpoint, "application/json", r)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	out, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return fmt.Errorf("%s: %s", resp.Status, strings.TrimSpace(string(out)))
	}
	var pretty any
	if json.Unmarshal(out, &pretty) == nil {
		pp, _ := json.MarshalIndent(pretty, "", "  ")
		fmt.Println(string(pp))
		return nil
	}
	fmt.Println(string(out))
	return nil
}
