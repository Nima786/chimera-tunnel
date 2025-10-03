package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// Config struct defines the structure for our JSON configuration file.
type Config struct {
	HandshakeMethod string `json:"handshake_method"`
	ListenAddress   string `json:"listen_address,omitempty"`  // Used by the server
	ConnectAddress  string `json:"connect_address,omitempty"` // Used by the client
	ProjectID       string `json:"project_id,omitempty"`      // For Google handshake
	TopicID         string `json:"topic_id,omitempty"`        // For Google handshake
}

func main() {
	// The program now takes a single, required argument: the path to its config file.
	configPath := flag.String("config", "", "Path to the JSON configuration file")
	flag.Parse()

	if *configPath == "" {
		log.Fatal("Error: -config flag is required. Please provide the path to a config file.")
	}

	// Load the configuration from the specified JSON file.
	config, err := loadConfig(*configPath)
	if err != nil {
		log.Fatalf("Error loading configuration from %s: %v", *configPath, err)
	}

	var sessionKey *[KeySize]byte
	var remoteAddr *net.UDPAddr

	// --- Main Logic: Now driven by the contents of the config file ---
	if config.ListenAddress != "" {
		// SERVER/LISTENER MODE
		fmt.Println("Starting Chimera in LISTEN mode...")
		if config.HandshakeMethod == "google" {
			var remoteAddrStr string
			sessionKey, remoteAddrStr, err = performGoogleHandshakeServer(config.ProjectID, config.TopicID)
			if err == nil {
				remoteAddr, err = net.ResolveUDPAddr("udp", remoteAddrStr)
			}
		} else { // Default to static
			sessionKey, remoteAddr, err = performStaticHandshakeServer(config.ListenAddress)
		}
		if err != nil {
			log.Fatalf("Handshake failed: %v", err)
		}
		fmt.Println("? Handshake successful! Starting data transport listener...")
		listenForData(sessionKey, config.ListenAddress, remoteAddr)

	} else if config.ConnectAddress != "" {
		// CLIENT/CONNECT MODE
		fmt.Println("Starting Chimera in CONNECT mode...")
		if config.HandshakeMethod == "google" {
			sessionKey, err = performGoogleHandshakeClient(config.ProjectID, config.TopicID, config.ConnectAddress)
		} else { // Default to static
			sessionKey, err = performStaticHandshakeClient(config.ConnectAddress)
		}
		if err != nil {
			log.Fatalf("Handshake failed: %v", err)
		}
		fmt.Println("? Handshake successful! Ready to send data.")
		runClientDataLoop(sessionKey, config.ConnectAddress)

	} else {
		log.Fatal("Invalid configuration: JSON file must specify either 'listen_address' or 'connect_address'")
	}
}

// loadConfig reads and parses the JSON configuration file.
func loadConfig(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var config Config
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		return nil, err
	}
	return &config, nil
}

// listenForData is the server's main loop after a successful handshake.
func listenForData(sessionKey *[KeySize]byte, listenAddrStr string, expectedRemoteAddr *net.UDPAddr) {
	udpAddr, err := net.ResolveUDPAddr("udp", listenAddrStr)
	if err != nil {
		log.Fatalf("Invalid listen address for data: %v", err)
	}
	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		log.Fatalf("Failed to start data listener: %v", err)
	}
	defer conn.Close()
	fmt.Printf("Listening for data on %s\n", listenAddrStr)

	buffer := make([]byte, 2048)
	for {
		n, remoteAddr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			log.Printf("Error reading data: %v", err)
			continue
		}

		decrypted, err := Decrypt(sessionKey, buffer[:n])
		if err != nil {
			log.Printf("Decryption failed from %s: %v", remoteAddr, err)
			continue
		}
		fmt.Printf("Received message: \"%s\"\n", string(decrypted))
	}
}

// runClientDataLoop is the client's main loop after a successful handshake.
func runClientDataLoop(sessionKey *[KeySize]byte, remoteAddrStr string) {
	conn, err := net.Dial("udp", remoteAddrStr)
	if err != nil {
		log.Fatalf("Failed to connect for data transfer: %v", err)
	}
	defer conn.Close()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	fmt.Println("Sending a test message every 2 seconds. Press Ctrl+C to stop.")

	for {
		select {
		case <-ticker.C:
			message := []byte(fmt.Sprintf("The time is %s", time.Now().Format(time.RFC3339)))
			encrypted, err := Encrypt(sessionKey, message)
			if err != nil {
				log.Printf("Encryption failed: %v", err)
				continue
			}
			if _, err := conn.Write(encrypted); err != nil {
				log.Printf("Failed to send message: %v", err)
			}
			fmt.Println("Sent encrypted message.")
		case <-sigChan:
			fmt.Println("\nSignal received, shutting down client.")
			return
		}
	}
}

// --- Handshake logic ---
func performStaticHandshakeServer(listenAddr string) (*[KeySize]byte, *net.UDPAddr, error) {
	fmt.Println("?? Starting Static handshake (Listen Mode)...")
	udpAddr, err := net.ResolveUDPAddr("udp", listenAddr)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid listen address: %w", err)
	}
	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to listen: %w", err)
	}
	defer conn.Close()

	buffer := make([]byte, 2048)
	fmt.Println("Waiting for a client handshake...")

	n, remoteAddr, err := conn.ReadFromUDP(buffer)
	if err != nil {
		return nil, nil, fmt.Errorf("handshake read failed: %w", err)
	}
	clientPubKey, err := curve.NewPublicKey(buffer[:n])
	if err != nil {
		return nil, nil, fmt.Errorf("invalid client public key: %v", err)
	}

	serverPrivKey, err := GenerateKeys()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate server keys: %v", err)
	}

	if _, err := conn.WriteTo(serverPrivKey.PublicKey().Bytes(), remoteAddr); err != nil {
		return nil, nil, fmt.Errorf("failed to send handshake reply: %v", err)
	}

	sessionKey, err := CalculateSharedSecret(serverPrivKey, clientPubKey)
	if err != nil {
		return nil, nil, err
	}
	return sessionKey, remoteAddr, nil
}

func performStaticHandshakeClient(connectAddr string) (*[KeySize]byte, error) {
	fmt.Println("?? Starting Static handshake (Connect Mode)...")
	conn, err := net.Dial("udp", connectAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}
	defer conn.Close()

	clientPrivKey, err := GenerateKeys()
	if err != nil {
		return nil, fmt.Errorf("failed to generate client keys: %w", err)
	}

	if _, err := conn.Write(clientPrivKey.PublicKey().Bytes()); err != nil {
		return nil, fmt.Errorf("failed to send handshake: %w", err)
	}

	buffer := make([]byte, 2048)
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	n, err := conn.Read(buffer)
	if err != nil {
		return nil, fmt.Errorf("failed to receive handshake reply: %v", err)
	}
	serverPubKey, err := curve.NewPublicKey(buffer[:n])
	if err != nil {
		return nil, fmt.Errorf("invalid server public key: %v", err)
	}

	return CalculateSharedSecret(clientPrivKey, serverPubKey)
}