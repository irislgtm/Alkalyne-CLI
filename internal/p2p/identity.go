package p2p

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
)

const identityFileName = "identity.key"
const identityFilePerms os.FileMode = 0600

func IdentityPath(dir string) string {
	return filepath.Join(dir, identityFileName)
}

func LoadOrCreateIdentity(path string) (crypto.PrivKey, error) {
	if data, err := os.ReadFile(path); err == nil {
		return crypto.UnmarshalPrivateKey(data)
	}

	priv, _, err := crypto.GenerateKeyPair(crypto.Ed25519, 0)
	if err != nil {
		return nil, fmt.Errorf("p2p: generate key: %w", err)
	}

	data, err := crypto.MarshalPrivateKey(priv)
	if err != nil {
		return nil, fmt.Errorf("p2p: marshal key: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, fmt.Errorf("p2p: mkdir: %w", err)
	}

	if err := os.WriteFile(path, data, identityFilePerms); err != nil {
		return nil, fmt.Errorf("p2p: write key: %w", err)
	}

	return priv, nil
}

func PeerIDFromPrivateKey(priv crypto.PrivKey) (string, error) {
	pid, err := peer.IDFromPrivateKey(priv)
	if err != nil {
		return "", fmt.Errorf("p2p: peer id: %w", err)
	}
	return pid.String(), nil
}
