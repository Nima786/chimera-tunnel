package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
)

type Config struct {
	Mode           string `json:"mode"`             // "relay" or "client"
	ListenAddress  string `json:"listen_address"`   // relay: TCP listen addr
	ConnectAddress string `json:"connect_address"`  // client: relay UDP addr
	DestAddress    string `json:"dest_address"`     // client: final destination (e.g. 127.0.0.1:8080)
}

func loadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func main() {
	configPath := flag.String("config", "/etc/chimera/server.json", "Path to JSON config")
	flag.Parse()

	cfg, err := loadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	switch cfg.Mode {
	case "relay":
		if cfg.ListenAddress == "" || cfg.ConnectAddress == "" {
			log.Fatal("Relay mode requires listen_address and connect_address (UDP of client)")
		}
		relay, err := NewRelay(cfg.ListenAddress, cfg.ConnectAddress)
		if err != nil {
			log.Fatalf("Failed to start relay: %v", err)
		}
		log.Printf("Running in RELAY mode: TCP listen %s -> UDP tunnel %s\n", cfg.ListenAddress, cfg.ConnectAddress)
		if err := relay.Run(); err != nil {
			log.Fatal(err)
		}

	case "client":
		if cfg.ConnectAddress == "" || cfg.DestAddress == "" {
			log.Fatal("Client mode requires connect_address (relay UDP) and dest_address (final)")
		}
		client, err := NewClient(":0", cfg.ConnectAddress, cfg.DestAddress)
		if err != nil {
			log.Fatalf("Failed to start client: %v", err)
		}
		log.Printf("Running in CLIENT mode: tunnel -> TCP %s\n", cfg.DestAddress)
		client.Run()

	default:
		fmt.Println("Invalid mode. Must be 'relay' or 'client'")
	}
}
