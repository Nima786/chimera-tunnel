package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// --- Configuration for Google Handshake (placeholders) ---
const projectID = "YOUR_PROJECT_ID" // <-- REMEMBER TO REPLACE THIS
const topicID = "chimera-rendezvous"

func main() {
	// --- Command-Line Flags ---
	handshakeMethod := flag.String("handshake", "static", "Handshake method: 'static' or 'google'")
	listenMode := flag.Bool("listen", false, "Run in server/listen mode")
	connectAddr := flag.String("connect", "", "Address for handshake/connection (e.g., '1.2.3.4:8080')")
	flag.Parse()

	var sessionKey *[KeySize]byte
	var remoteAddr *net.UDPAddr
	var err error

	// --- Main Logic: Perform Handshake and Start Tunnel ---
	if *listenMode {
		// SERVER/LISTENER MODE
		fmt.Println("Starting Chimera in LISTEN mode...")
		if *handshakeMethod == "google" {
			var remoteAddrStr string
			sessionKey, remoteAddrStr, err = performGoogleHandshakeServer(projectID, topicID)
			if err == nil {
				remoteAddr, err = net.ResolveUDPAddr("udp", remoteAddrStr)
			}
		} else {
			sessionKey, remoteAddr, err = performStaticHandshakeServer(*connectAddr)
		}
		if err != nil {
			log.Fatalf("Handshake failed: %v", err)
		}
		fmt.Println("? Handshake successful! Starting data transport listener...")
		// Pass the original listen address AND the expected client address
		listenForData(sessionKey, *connectAddr, remoteAddr)

	} else if *connectAddr != "" {
		// CLIENT/CONNECT MODE
		fmt.Println("Starting Chimera in CONNECT mode...")
		var remoteAddrStr string
		if *handshakeMethod == "google" {
			sessionKey, err = performGoogleHandshakeClient(projectID, topicID, *connectAddr)
			remoteAddrStr = *connectAddr
		} else {
			sessionKey, err = performStaticHandshakeClient(*connectAddr)
			remoteAddrStr = *connectAddr
		}
		if err != nil {
			log.Fatalf("Handshake failed: %v", err)
		}
		fmt.Println("? Handshake successful! Ready to send data.")
		runClientDataLoop(sessionKey, remoteAddrStr)

	} else {
		fmt.Println("Usage: go run . -listen -connect <ip:port> OR go run . -connect <ip:port>")
		flag.PrintDefaults()
	}
} // <-- THIS BRACE WAS IN THE WRONG PLACE

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