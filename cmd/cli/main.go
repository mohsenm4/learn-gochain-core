package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
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
	}
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
		fmt.Fprintf(os.Stderr, "  %-22s %s\n", cmd.usage, cmd.summary)
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
