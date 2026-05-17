package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/alkalyne/alkalyne/internal/config"
	"github.com/alkalyne/alkalyne/internal/db"
	"github.com/alkalyne/alkalyne/internal/models"
	"github.com/alkalyne/alkalyne/internal/p2p"
	"github.com/alkalyne/alkalyne/internal/tui"
)

func main() {
	dataDir := flag.String("data-dir", "", "data directory (default: ~/.alkalyne)")
	port := flag.Int("port", 9000, "listen port")
	relayMode := flag.Bool("relay", false, "act as a circuit relay for other peers")
	noTUI := flag.Bool("no-tui", false, "disable TUI (pipe-friendly mode)")
	help := flag.Bool("help", false, "print usage")
	flag.Parse()

	if *help {
		printUsage()
		return
	}

	args := flag.Args()
	mode := "client"
	if len(args) > 0 {
		switch args[0] {
		case "daemon":
			mode = "daemon"
		case "relay-setup":
			mode = "relay-setup"
		default:
			printUsage()
			os.Exit(1)
		}
	}

	cfgPath := *dataDir
	if cfgPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			log.Fatalf("cannot determine home dir: %v", err)
		}
		cfgPath = home + "/.alkalyne"
	}

	cfgFile := cfgPath + "/config.toml"
	cfg, err := config.Load(cfgFile)
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	if *dataDir != "" {
		cfg.DataDir = *dataDir
	}
	if *port != 9000 {
		cfg.ListenAddrs = []string{fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", *port)}
	}

	switch mode {
	case "client":
		runClient(cfg, cfgFile, *noTUI, *relayMode)

	case "daemon":
		fmt.Println("daemon mode not yet implemented")
		os.Exit(0)

	case "relay-setup":
		fmt.Println("relay-setup mode not yet implemented")
		os.Exit(0)
	}
}

func runClient(cfg *models.Config, cfgPath string, noTUI bool, relayMode bool) {
	dataDir := config.DataDir(cfg)
	identityPath := p2p.IdentityPath(dataDir)
	privKey, err := p2p.LoadOrCreateIdentity(identityPath)
	if err != nil {
		log.Fatalf("identity: %v", err)
	}

	peerID, err := p2p.PeerIDFromPrivateKey(privKey)
	if err != nil {
		log.Fatalf("peer id: %v", err)
	}

	h, err := p2p.NewHost(privKey, cfg.ListenAddrs, relayMode)
	if err != nil {
		log.Fatalf("p2p host: %v", err)
	}
	defer func() { _ = h.Close() }()

	ctx := context.Background()

	if len(cfg.BootstrapPeers) > 0 {
		errs := p2p.ConnectToPeers(ctx, h, cfg.BootstrapPeers)
		for _, e := range errs {
			log.Printf("bootstrap: %v", e)
		}
	}

	ps, err := p2p.NewPubSub(ctx, h)
	if err != nil {
		log.Fatalf("pubsub: %v", err)
	}

	lobbyTopic, lobbySub, err := p2p.JoinTopic(ps, p2p.LobbyTopic)
	if err != nil {
		log.Fatalf("lobby: %v", err)
	}

	disc := p2p.NewDiscovery(h)
	if err := disc.Start(); err != nil {
		log.Printf("discovery: %v", err)
	}
	defer func() { _ = disc.Close() }()

	dbPath := filepath.Join(dataDir, "alkalyne.db")
	database, err := db.Open(dbPath)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer func() { _ = database.Close() }()

	if noTUI {
		fmt.Println("pipe mode not yet implemented")
		os.Exit(0)
	}

	be := &tui.Backend{
		Host:       h,
		PubSub:     ps,
		LobbyTopic: lobbyTopic,
		LobbySub:   lobbySub,
		LobbyCtx:   ctx,
		DB:         database,
		PeerID:     peerID,
		Config:     cfg,
		ConfigPath: cfgPath,
	}

	if err := tui.Start(be); err != nil {
		log.Fatalf("tui: %v", err)
	}
}

func printUsage() {
	fmt.Println(`Alkalyne — P2P messaging CLI

Usage:
  alkalyne                        Start in client (TUI) mode
  alkalyne daemon                 Start as headless relay node
  alkalyne relay-setup            Run the relay configuration wizard

Flags:
  --data-dir <path>  Data directory (default: ~/.alkalyne)
  --port <n>         Listen port (default: 9000)
  --relay            Act as a circuit relay for other peers
  --no-tui           Disable TUI, print to stdout
  --help             Print this help`)
}
