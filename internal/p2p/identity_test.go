package p2p

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCreateIdentity(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, identityFileName)

	priv, err := LoadOrCreateIdentity(keyPath)
	if err != nil {
		t.Fatalf("LoadOrCreateIdentity: %v", err)
	}
	if priv == nil {
		t.Fatal("expected non-nil private key")
	}

	pid, err := PeerIDFromPrivateKey(priv)
	if err != nil {
		t.Fatalf("PeerIDFromPrivateKey: %v", err)
	}
	if pid == "" {
		t.Fatal("expected non-empty peer ID")
	}
}

func TestLoadExistingIdentity(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, identityFileName)

	orig, err := LoadOrCreateIdentity(keyPath)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	loaded, err := LoadOrCreateIdentity(keyPath)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	origRaw, err := orig.Raw()
	if err != nil {
		t.Fatalf("orig raw: %v", err)
	}
	loadedRaw, err := loaded.Raw()
	if err != nil {
		t.Fatalf("loaded raw: %v", err)
	}

	if string(origRaw) != string(loadedRaw) {
		t.Fatal("loaded key does not match original")
	}
}

func TestIdentityFilePermissions(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, identityFileName)

	_, err := LoadOrCreateIdentity(keyPath)
	if err != nil {
		t.Fatalf("LoadOrCreateIdentity: %v", err)
	}

	info, err := os.Stat(keyPath)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}

	if info.Mode() != identityFilePerms {
		t.Errorf("expected permissions %o, got %o", identityFilePerms, info.Mode())
	}
}

func TestIdentityPath(t *testing.T) {
	path := IdentityPath("/tmp/alkalyne")
	expected := "/tmp/alkalyne/identity.key"
	if path != expected {
		t.Errorf("expected %s, got %s", expected, path)
	}
}

func TestPeerIDDeterministic(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, identityFileName)

	priv, err := LoadOrCreateIdentity(keyPath)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	pid1, err := PeerIDFromPrivateKey(priv)
	if err != nil {
		t.Fatalf("first call: %v", err)
	}

	pid2, err := PeerIDFromPrivateKey(priv)
	if err != nil {
		t.Fatalf("second call: %v", err)
	}

	if pid1 != pid2 {
		t.Fatal("PeerID must be deterministic for same key")
	}
}

func TestLoadMissingFile(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "subdir", identityFileName)

	_, err := LoadOrCreateIdentity(keyPath)
	if err != nil {
		t.Fatalf("LoadOrCreateIdentity on missing path: %v", err)
	}

	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		t.Fatal("identity file was not created")
	}
}
