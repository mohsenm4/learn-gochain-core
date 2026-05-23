package main

import (
	"context"

	"github.com/Mohsen20031203/learn-gochain-core/config"
	api "github.com/Mohsen20031203/learn-gochain-core/internal/api/http"
	"github.com/Mohsen20031203/learn-gochain-core/internal/api/http/handler"
	"github.com/Mohsen20031203/learn-gochain-core/internal/infrastructure/network"
	"github.com/Mohsen20031203/learn-gochain-core/internal/usecase/blockchain"
)

func main() {

	cfg, err := config.LoadConfig(".")
	if err != nil {
		panic(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Usecase
	nodeService := blockchain.NewService(cfg)

	if err := nodeService.BootstrapGenesis(); err != nil {
		panic(err)
	}

	tcpServer := network.NewTCPServer(
		cfg.TCPAddress,
		nodeService.HandleNodeMessage,
	)

	if err := tcpServer.Start(); err != nil {
		panic(err)
	}

	// 🔹 Gossiper
	gossiper := network.NewTCPGossiper(cfg.Peers, cfg.TCPAddress)
	nodeService.SetGossiper(gossiper)
	nodeService.AnnouncePeers()
	nodeService.RequestChain()

	h := handler.NewHandler(nodeService)
	server := api.NewServer(cfg, h)

	// 🔹 Miner
	nodeService.StartMiner(ctx)

	if err := server.Start(); err != nil {
		panic(err)
	}

}
