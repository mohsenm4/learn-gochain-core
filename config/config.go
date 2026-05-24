package config

import (
	"errors"
	"fmt"
	"net"

	"github.com/spf13/viper"
)

type Config struct {
	Port            string   `mapstructure:"API_PORT"`
	Difficulty      int      `mapstructure:"BLOCKCHAIN_DIFFICULTY"`
	FileStoragePath string   `mapstructure:"FILE_STORAGE_PATH"`
	NodeID          string   `mapstructure:"NODE_ID"`
	BatchSize       int      `mapstructure:"BATCH_SIZE"`
	Peers           []string `mapstructure:"PEERS"`
	TCPAddress      string   `mapstructure:"TCP_PORT"`
	PublicAddress   string   `mapstructure:"PUBLIC_ADDR"`
	MinerWalletPath string   `mapstructure:"MINER_WALLET_PATH"`
}

func LoadConfig(path string) (config Config, err error) {
	viper.AddConfigPath(path)
	viper.SetConfigName("app")
	viper.SetConfigType("env")

	viper.SetDefault("API_PORT", "9090")
	viper.SetDefault("BLOCKCHAIN_DIFFICULTY", 3)
	viper.SetDefault("FILE_STORAGE_PATH", "chainDB")
	viper.SetDefault("BATCH_SIZE", 1)
	viper.SetDefault("NODE_ID", "node")
	viper.SetDefault("TCP_PORT", "0.0.0.0:7000")
	viper.SetDefault("PEERS", "")
	viper.SetDefault("MINER_WALLET_PATH", "miner.wallet.json")
	viper.SetDefault("PUBLIC_ADDR", "")

	viper.AutomaticEnv()

	if err = viper.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if !errors.As(err, &notFound) {
			return
		}
		err = nil
	}

	if err = viper.Unmarshal(&config); err != nil {
		return
	}

	if config.PublicAddress == "" {
		_, port, splitErr := net.SplitHostPort(config.TCPAddress)
		if splitErr == nil && config.NodeID != "" {
			config.PublicAddress = config.NodeID + ":" + port
		} else {
			config.PublicAddress = config.TCPAddress
		}
	}

	for i := len(config.Peers) - 1; i >= 0; i-- {
		if config.Peers[i] == config.PublicAddress {
			config.Peers = append(config.Peers[:i], config.Peers[i+1:]...)
		}
	}

	fmt.Println("node:", config.NodeID, "api:", config.Port, "tcp:", config.TCPAddress, "public:", config.PublicAddress, "peers:", config.Peers)
	return
}
