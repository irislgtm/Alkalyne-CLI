package mailbox

import (
	"testing"
)

func TestEncryptDecryptRoundtrip(t *testing.T) {
	t.Parallel()

	keyPair, err := GenerateEncryptionKeyPair()
	if err != nil {
		t.Fatalf("generate keypair: %v", err)
	}

	plaintext := []byte("secret message for offline peer")

	ciphertext, err := EncryptMessage(plaintext, &keyPair.PublicKey)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	if len(ciphertext) <= 32+24 {
		t.Fatal("ciphertext too short")
	}

	decrypted, err := DecryptMessage(ciphertext, &keyPair.PrivateKey)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}

	if !VerifyDecryption(decrypted, plaintext) {
		t.Errorf("decrypted = %q, want %q", decrypted, plaintext)
	}
}

func TestDecryptWithWrongKey(t *testing.T) {
	t.Parallel()

	keyPair, err := GenerateEncryptionKeyPair()
	if err != nil {
		t.Fatalf("generate keypair: %v", err)
	}

	wrongPair, err := GenerateEncryptionKeyPair()
	if err != nil {
		t.Fatalf("generate wrong keypair: %v", err)
	}

	plaintext := []byte("secret message")

	ciphertext, err := EncryptMessage(plaintext, &keyPair.PublicKey)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	_, err = DecryptMessage(ciphertext, &wrongPair.PrivateKey)
	if err == nil {
		t.Fatal("expected decryption failure with wrong key")
	}
}

func TestDecryptTooShort(t *testing.T) {
	t.Parallel()

	keyPair, err := GenerateEncryptionKeyPair()
	if err != nil {
		t.Fatalf("generate keypair: %v", err)
	}

	_, err = DecryptMessage([]byte("short"), &keyPair.PrivateKey)
	if err == nil {
		t.Fatal("expected error for short ciphertext")
	}
}

func TestEncryptProducesDifferentCiphertext(t *testing.T) {
	t.Parallel()

	keyPair, err := GenerateEncryptionKeyPair()
	if err != nil {
		t.Fatalf("generate keypair: %v", err)
	}

	plaintext := []byte("same message")

	ct1, err := EncryptMessage(plaintext, &keyPair.PublicKey)
	if err != nil {
		t.Fatalf("encrypt 1: %v", err)
	}

	ct2, err := EncryptMessage(plaintext, &keyPair.PublicKey)
	if err != nil {
		t.Fatalf("encrypt 2: %v", err)
	}

	if string(ct1) == string(ct2) {
		t.Fatal("ciphertext should differ per encryption due to ephemeral key and nonce")
	}

	d1, err := DecryptMessage(ct1, &keyPair.PrivateKey)
	if err != nil {
		t.Fatalf("decrypt 1: %v", err)
	}

	d2, err := DecryptMessage(ct2, &keyPair.PrivateKey)
	if err != nil {
		t.Fatalf("decrypt 2: %v", err)
	}

	if !VerifyDecryption(d1, d2) {
		t.Fatal("decrypted messages should match")
	}
}

func TestGenerateEncryptionKeyPair(t *testing.T) {
	t.Parallel()

	kp1, err := GenerateEncryptionKeyPair()
	if err != nil {
		t.Fatalf("generate keypair 1: %v", err)
	}

	kp2, err := GenerateEncryptionKeyPair()
	if err != nil {
		t.Fatalf("generate keypair 2: %v", err)
	}

	if kp1.PublicKey == kp2.PublicKey {
		t.Fatal("keypairs should be unique")
	}
}
