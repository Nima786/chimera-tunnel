package main

import (
	"crypto/rand"
	"fmt"
	"io"

	"golang.org/x/crypto/curve25519"
)

// Key exchange keys are 32 bytes, just like our encryption keys.
const handshakeKeySize = 32

// GenerateKeys creates a new public/private key pair for the handshake.
func GenerateKeys() (publicKey, privateKey *[handshakeKeySize]byte, err error) {
	// Create space in memory for the keys.
	var privKey [handshakeKeySize]byte
	var pubKey [handshakeKeySize]byte

	// Generate a new random private key.
	if _, err := io.ReadFull(rand.Reader, privKey[:]); err != nil {
		return nil, nil, fmt.Errorf("could not generate private key: %w", err)
	}

	// Derive the public key from the private key.
	curve25519.ScalarBase(&pubKey, &privKey)

	return &pubKey, &privKey, nil
}

// CalculateSharedSecret performs the Diffie-Hellman calculation.
// It takes our own private key and the other party's public key.
// It returns a 32-byte shared secret that can be used as our encryption key.
func CalculateSharedSecret(privateKey, publicKey *[handshakeKeySize]byte) (*[KeySize]byte, error) {
	var sharedSecret [KeySize]byte

	// The core of the key exchange. The result is a secret that only the two
	// parties who own the corresponding private keys can calculate.
	curve25519.ScalarMult(&sharedSecret, privateKey, publicKey)

	return &sharedSecret, nil
}
