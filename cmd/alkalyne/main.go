package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/alkalyne/alkalyne/internal/config"
	"github.com/alkalyne/alkalyne/internal/db"
	"github.com/alkalyne/alkalyne/internal/mailbox"
	"github.com/alkalyne/alkalyne/internal/models"
	"github.com/alkalyne/alkalyne/internal/p2p"
	"github.com/alkalyne/alkalyne/internal/tui"
	"github.com/libp2p/go-libp2p/p2p/discovery/routing"
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
		runDaemon(cfg)

	case "relay-setup":
		runRelaySetup()
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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	if len(cfg.BootstrapPeers) > 0 {
		log.Printf("connecting to %d bootstrap peers...", len(cfg.BootstrapPeers))
		errs := p2p.ConnectToPeers(ctx, h, cfg.BootstrapPeers)
		for _, e := range errs {
			log.Printf("bootstrap: %v", e)
		}
	}

	dhtInstance, err := p2p.SetupDHT(ctx, h)
	if err != nil {
		log.Fatalf("dht: %v", err)
	}
	defer func() { _ = dhtInstance.Close() }()

	if err := p2p.BootstrapDHT(ctx, dhtInstance); err != nil {
		log.Printf("dht bootstrap: %v", err)
	}

	disc := routing.NewRoutingDiscovery(dhtInstance)
	ps, err := p2p.NewPubSubWithDiscovery(ctx, h, disc)
	if err != nil {
		log.Fatalf("pubsub: %v", err)
	}

	lobbyTopic, lobbySub, err := p2p.JoinTopic(ps, p2p.LobbyTopic)
	if err != nil {
		log.Fatalf("lobby: %v", err)
	}

	mDNS := p2p.NewDiscovery(h)
	if err := mDNS.Start(); err != nil {
		log.Printf("mdns: %v", err)
	}
	defer func() { _ = mDNS.Close() }()

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

func runDaemon(cfg *models.Config) {
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

	// daemon always acts as a circuit relay and mailbox relay
	h, err := p2p.NewHost(privKey, cfg.ListenAddrs, true)
	if err != nil {
		log.Fatalf("p2p host: %v", err)
	}
	defer func() { _ = h.Close() }()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	peerIDStr := peerID
	log.Printf("alkalyne daemon starting (peer: %s)", peerIDStr)
	log.Printf("listening on: %v", h.Addrs())
	for _, addr := range h.Addrs() {
		log.Printf("  %s/p2p/%s", addr, peerIDStr)
	}

	if len(cfg.BootstrapPeers) > 0 {
		log.Printf("connecting to %d bootstrap peers...", len(cfg.BootstrapPeers))
		errs := p2p.ConnectToPeers(ctx, h, cfg.BootstrapPeers)
		for _, e := range errs {
			log.Printf("bootstrap: %v", e)
		}
	}

	if cfg.DHTEnabled {
		log.Print("initializing DHT...")
		dhtInstance, err := p2p.SetupDHT(ctx, h)
		if err != nil {
			log.Printf("dht: %v", err)
		} else {
			defer func() { _ = dhtInstance.Close() }()
			if err := p2p.BootstrapDHT(ctx, dhtInstance); err != nil {
				log.Printf("dht bootstrap: %v", err)
			}
		}
	}

	mDNS := p2p.NewDiscovery(h)
	if err := mDNS.Start(); err != nil {
		log.Printf("mdns: %v", err)
	}
	defer func() { _ = mDNS.Close() }()

	mbStore := mailbox.NewStore()
	mbRelay := mailbox.NewRelay(h, mbStore)
	if err := mbRelay.Start(); err != nil {
		log.Fatalf("mailbox relay: %v", err)
	}
	log.Print("mailbox relay ready (protocol: /alkalyne/mailbox/1.0.0)")

	log.Print("daemon running. press Ctrl+C to stop.")
	<-sigCh
	log.Print("shutting down...")
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
