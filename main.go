package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"time"
)

func main() {
	listenMode := flag.Bool("listen", false, "Run in server/listen mode")
	connectAddr := flag.String("connect", "", "Address to connect to (e.g., '127.0.0.1:8080')")
	flag.Parse()

	if *listenMode {
		runServer()
	} else if *connectAddr != "" {
		runClient(*connectAddr)
	} else {
		fmt.Println("Usage: go run . -listen OR go run . -connect <ip:port>")
		flag.PrintDefaults()
	}
}

func runServer() {
	addr := "127.0.0.1:8080"
	fmt.Printf("?? Starting Chimera server in listen mode on %s...\n", addr)

	conn, err := net.ListenPacket("udp", addr)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	defer conn.Close()

	buffer := make([]byte, 2048)

	// --- Server Handshake Logic ---
	fmt.Println("Waiting for a client handshake...")

	// 1. Wait for the client's public key.
	n, remoteAddr, err := conn.ReadFrom(buffer)
	if err != nil {
		log.Fatalf("Handshake read failed: %v", err)
	}
	if n != handshakeKeySize {
		log.Fatalf("Invalid handshake from %s: incorrect key size", remoteAddr)
	}
	var clientPubKey [handshakeKeySize]byte
	copy(clientPubKey[:], buffer[:n])
	fmt.Printf("Received handshake from %s\n", remoteAddr)

	// 2. Generate our own key pair.
	serverPubKey, serverPrivKey, err := GenerateKeys()
	if err != nil {
		log.Fatalf("Failed to generate server keys: %v", err)
	}

	// 3. Send our public key back to the client.
	if _, err := conn.WriteTo(serverPubKey[:], remoteAddr); err != nil {
		log.Fatalf("Failed to send handshake reply: %v", err)
	}
	fmt.Println("Sent handshake reply.")

	// 4. Calculate the shared secret session key.
	sessionKey, err := CalculateSharedSecret(serverPrivKey, &clientPubKey)
	if err != nil {
		log.Fatalf("Failed to calculate shared secret: %v", err)
	}
	fmt.Println("? Secure session established!")

	// --- Listen for Encrypted Data ---
	for {
		n, _, err := conn.ReadFrom(buffer)
		if err != nil {
			log.Printf("Error reading data: %v", err)
			continue
		}
		decrypted, err := Decrypt(sessionKey, buffer[:n])
		if err != nil {
			log.Printf("Decryption failed: %v", err)
			continue
		}
		fmt.Printf("Received message: \"%s\"\n", string(decrypted))
	}
}

func runClient(connectAddr string) {
	fmt.Printf("?? Starting Chimera client, connecting to %s...\n", connectAddr)

	conn, err := net.Dial("udp", connectAddr)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	// --- Client Handshake Logic ---
	// 1. Generate our key pair.
	clientPubKey, clientPrivKey, err := GenerateKeys()
	if err != nil {
		log.Fatalf("Failed to generate client keys: %v", err)
	}

	// 2. Send our public key to the server to initiate the handshake.
	if _, err := conn.Write(clientPubKey[:]); err != nil {
		log.Fatalf("Failed to send handshake: %v", err)
	}
	fmt.Println("Sent handshake, waiting for reply...")

	// 3. Wait for the server's public key.
	buffer := make([]byte, 2048)
	// Set a deadline so we don't wait forever if the server is down.
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	n, err := conn.Read(buffer)
	if err != nil {
		log.Fatalf("Failed to receive handshake reply: %v", err)
	}
	if n != handshakeKeySize {
		log.Fatalf("Invalid handshake reply: incorrect key size")
	}
	var serverPubKey [handshakeKeySize]byte
	copy(serverPubKey[:], buffer[:n])

	// 4. Calculate the shared secret session key.
	sessionKey, err := CalculateSharedSecret(clientPrivKey, &serverPubKey)
	if err != nil {
		log.Fatalf("Failed to calculate shared secret: %v", err)
	}
	fmt.Println("? Secure session established!")

	// --- Send Encrypted Data ---
	message := []byte("This message is protected by a dynamic session key!")
	encrypted, err := Encrypt(sessionKey, message)
	if err != nil {
		log.Fatalf("Encryption failed: %v", err)
	}

	if _, err := conn.Write(encrypted); err != nil {
		log.Fatalf("Failed to send message: %v", err)
	}
	fmt.Println("Sent encrypted message.")
}