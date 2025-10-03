package main

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/nacl/secretbox"
)

// NonceSize is the size of the nonce for ChaCha20-Poly1305.
const NonceSize = 24

// KeySize is the size of the key for ChaCha20-Poly1305.
const KeySize = 32

// Encrypt encrypts a message using a key and a new, random nonce.
// The nonce is prepended to the ciphertext for use during decryption.
func Encrypt(key *[KeySize]byte, message []byte) ([]byte, error) {
	var nonce [NonceSize]byte
	if _, err := io.ReadFull(rand.Reader, nonce[:]); err != nil {
		return nil, fmt.Errorf("could not generate nonce: %w", err)
	}

	encrypted := secretbox.Seal(nonce[:], message, &nonce, key)
	return encrypted, nil
}

// Decrypt decrypts a message using a key.
// It extracts the nonce from the beginning of the message.
func Decrypt(key *[KeySize]byte, encryptedMessage []byte) ([]byte, error) {
	if len(encryptedMessage) < NonceSize {
		return nil, errors.New("invalid encrypted message: too short")
	}

	var nonce [NonceSize]byte
	copy(nonce[:], encryptedMessage[:NonceSize])

	decrypted, ok := secretbox.Open(nil, encryptedMessage[NonceSize:], &nonce, key)
	if !ok {
		return nil, errors.New("decryption failed (invalid key or corrupted message)")
	}

	return decrypted, nil
}
