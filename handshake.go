package main

import (
	"crypto/ecdh"
	"crypto/rand"
	"fmt"
)

// The standard Curve25519 is used for the key exchange.
var curve = ecdh.X25519()

// GenerateKeys creates a new public/private key pair for the handshake.
func GenerateKeys() (*ecdh.PrivateKey, error) {
	// This function generates a new, random private key for the X25519 curve.
	// The public key is part of the private key object.
	privateKey, err := curve.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("could not generate private key: %w", err)
	}
	return privateKey, nil
}

// CalculateSharedSecret performs the Diffie-Hellman calculation.
// It takes our own private key and the other party's public key.
// It returns a 32-byte shared secret that can be used as our encryption key.
func CalculateSharedSecret(privateKey *ecdh.PrivateKey, publicKey *ecdh.PublicKey) (*[KeySize]byte, error) {
	// This is the core of the key exchange. It takes our private key and the peer's
	// public key and computes the shared secret.
	sharedSecretBytes, err := privateKey.ECDH(publicKey)
	if err != nil {
		return nil, fmt.Errorf("could not perform ECDH: %w", err)
	}

	// The result is a byte slice. We need to convert it to a fixed-size 32-byte array
	// to be compatible with our encryption function.
	var sharedSecret [KeySize]byte
	copy(sharedSecret[:], sharedSecretBytes)

	return &sharedSecret, nil
}
