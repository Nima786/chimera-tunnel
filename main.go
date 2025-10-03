package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"time"
)

// --- Configuration for Google Handshake ---
// We now load credentials from the "gcloud-key.json" file,
// so we only need the Project ID and Topic ID here.
const projectID = "chimera-handshake" // <-- REMEMBER TO REPLACE THIS
const topicID = "chimera-rendezvous"

func main() {
	// --- Command-Line Flags ---
	handshakeMethod := flag.String("handshake", "static", "Handshake method: 'static' or 'google'")
	listenMode := flag.Bool("listen", false, "Run in server/listen mode")
	connectAddr := flag.String("connect", "", "Address for handshake/connection (e.g., '127.0.0.1:8080')")
	flag.Parse()

	var sessionKey *[KeySize]byte
	var remoteAddrStr string
	var err error

	// --- Main Logic: Perform the chosen handshake ---
	if *listenMode {
		// SERVER MODE
		if *handshakeMethod == "google" {
			// Corrected: No longer passes apiKey
			sessionKey, remoteAddrStr, err = performGoogleHandshakeServer(projectID, topicID)
		} else {
			sessionKey, remoteAddrStr, err = performStaticHandshakeServer(*connectAddr)
		}
		if err != nil {
			log.Fatalf("Handshake failed: %v", err)
		}
		fmt.Println("? Handshake successful! Starting data listener...")
		// For this test, we just prove the handshake worked.
		fmt.Println("Milestone 3 (Server) Complete!")

	} else {
		// CLIENT MODE
		if *handshakeMethod == "google" {
			// Corrected: No longer passes apiKey
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
		sendTestData(sessionKey, remoteAddrStr)
	}
}

// This function sends an encrypted test message.
func sendTestData(sessionKey *[KeySize]byte, remoteAddrStr string) {
	conn, err := net.Dial("udp", remoteAddrStr)
	if err != nil {
		log.Fatalf("Failed to connect for data transfer: %v", err)
	}
	defer conn.Close()

	message := []byte("This message is protected by the new session key!")
	encrypted, err := Encrypt(sessionKey, message)
	if err != nil {
		log.Fatalf("Encryption failed: %v", err)
	}

	if _, err := conn.Write(encrypted); err != nil {
		log.Fatalf("Failed to send message: %v", err)
	}
	fmt.Println("Sent encrypted test message.")
}

// --- Static handshake logic ---
func performStaticHandshakeServer(listenAddr string) (*[KeySize]byte, string, error) {
	fmt.Println("?? Starting Static handshake (Listen Mode)...")
	conn, err := net.ListenPacket("udp", listenAddr)
	if err != nil {
		return nil, "", fmt.Errorf("failed to listen: %w", err)
	}
	defer conn.Close()

	buffer := make([]byte, 2048)
	fmt.Println("Waiting for a client handshake...")

	n, remoteAddr, err := conn.ReadFrom(buffer)
	if err != nil {
		return nil, "", fmt.Errorf("handshake read failed: %w", err)
	}
	clientPubKey, err := curve.NewPublicKey(buffer[:n])
	if err != nil {
		return nil, "", fmt.Errorf("invalid client public key: %v", err)
	}

	serverPrivKey, err := GenerateKeys()
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate server keys: %v", err)
	}

	if _, err := conn.WriteTo(serverPrivKey.PublicKey().Bytes(), remoteAddr); err != nil {
		return nil, "", fmt.Errorf("failed to send handshake reply: %v", err)
	}

	sessionKey, err := CalculateSharedSecret(serverPrivKey, clientPubKey)
	if err != nil {
		return nil, "", err
	}
	return sessionKey, remoteAddr.String(), nil
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