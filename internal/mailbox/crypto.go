package mailbox

import (
	"crypto/rand"
	"fmt"

	"golang.org/x/crypto/nacl/box"
)

type EncryptionKeyPair struct {
	PublicKey  [32]byte
	PrivateKey [32]byte
}

func GenerateEncryptionKeyPair() (*EncryptionKeyPair, error) {
	pub, priv, err := box.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("mailbox: generate keypair: %w", err)
	}
	return &EncryptionKeyPair{
		PublicKey:  *pub,
		PrivateKey: *priv,
	}, nil
}

func EncryptMessage(plaintext []byte, recipientPub *[32]byte) ([]byte, error) {
	ephemeralPub, ephemeralPriv, err := box.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("mailbox: generate ephemeral key: %w", err)
	}

	nonce := new([24]byte)
	if _, err := rand.Read(nonce[:]); err != nil {
		return nil, fmt.Errorf("mailbox: generate nonce: %w", err)
	}

	ciphertext := box.Seal(nil, plaintext, nonce, recipientPub, ephemeralPriv)

	result := make([]byte, 32+24+len(ciphertext))
	copy(result[:32], ephemeralPub[:])
	copy(result[32:56], nonce[:])
	copy(result[56:], ciphertext)

	return result, nil
}

func DecryptMessage(ciphertext []byte, recipientPriv *[32]byte) ([]byte, error) {
	if len(ciphertext) < 32+24 {
		return nil, fmt.Errorf("mailbox: ciphertext too short")
	}

	var ephemeralPub [32]byte
	copy(ephemeralPub[:], ciphertext[:32])

	var nonce [24]byte
	copy(nonce[:], ciphertext[32:56])

	plaintext, ok := box.Open(nil, ciphertext[56:], &nonce, &ephemeralPub, recipientPriv)
	if !ok {
		return nil, fmt.Errorf("mailbox: decryption failed")
	}

	return plaintext, nil
}

func VerifyDecryption(plaintext []byte, expected []byte) bool {
	return string(plaintext) == string(expected)
}
