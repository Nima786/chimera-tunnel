package main

import (
	"flag"
	"fmt"
	"log"
	"net"
)

func main() {
	// --- Command-Line Flags ---
	// We use flags to tell our program whether to run as a "listen" (server)
	// or "connect" (client).
	listenMode := flag.Bool("listen", false, "Run in server/listen mode")
	connectAddr := flag.String("connect", "", "Address to connect to (e.g., '127.0.0.1:8080')")
	flag.Parse()

	// --- Shared Secret Key ---
	// For this milestone, we will use a simple, hard-coded key.
	// In the next milestone, we will generate this dynamically.
	// IMPORTANT: This is just a placeholder key for testing!
	var key [KeySize]byte
	// We fill the key with a placeholder value. A real key would be random.
	for i := 0; i < KeySize; i++ {
		key[i] = byte(i)
	}

	// --- Main Logic ---
	if *listenMode {
		// If the -listen flag is used, run the server logic.
		runServer(&key)
	} else if *connectAddr != "" {
		// If the -connect flag is used, run the client logic.
		runClient(&key, *connectAddr)
	} else {
		// If no flags are given, print usage instructions.
		fmt.Println("Usage: go run . -listen OR go run . -connect <ip:port>")
		flag.PrintDefaults()
	}
}

// runServer contains the logic for the listener.
func runServer(key *[KeySize]byte) {
	addr := "127.0.0.1:8080"
	fmt.Printf("🚀 Starting Chimera server and listening on %s...\n", addr)

	// Listen for incoming UDP packets on our address.
	conn, err := net.ListenPacket("udp", addr)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	defer conn.Close()

	// Create a buffer to hold incoming data.
	buffer := make([]byte, 2048)

	for {
		// Wait for and read a packet from the network.
		n, remoteAddr, err := conn.ReadFrom(buffer)
		if err != nil {
			log.Printf("Error reading from connection: %v", err)
			continue
		}

		fmt.Printf("\nReceived %d encrypted bytes from %s\n", n, remoteAddr)

		// Decrypt the message using our protocol function.
		decrypted, err := Decrypt(key, buffer[:n])
		if err != nil {
			log.Printf("Decryption failed: %v", err)
			continue
		}

		fmt.Printf("✅ Decrypted message: \"%s\"\n", string(decrypted))
	}
}

// runClient contains the logic for the initiator.
func runClient(key *[KeySize]byte, connectAddr string) {
	fmt.Printf("🚀 Starting Chimera client, connecting to %s...\n", connectAddr)

	// Dial the server. This doesn't create a persistent connection for UDP,
	// but it sets up the destination address.
	conn, err := net.Dial("udp", connectAddr)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	message := []byte("Hello from the Chimera client!")

	// Encrypt the message using our protocol function.
	encrypted, err := Encrypt(key, message)
	if err != nil {
		log.Fatalf("Encryption failed: %v", err)
	}

	// Send the encrypted message to the server.
	_, err = conn.Write(encrypted)
	if err != nil {
		log.Fatalf("Failed to send message: %v", err)
	}

	fmt.Printf("✅ Sent encrypted message!\n")
}
