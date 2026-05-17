package models

type RelayConfig struct {
	PeerID  string   `toml:"peer_id"`
	Addrs   []string `toml:"addrs"`
	Enabled bool     `toml:"enabled"`
}

type Theme struct {
	Name       string `toml:"name"`
	Background string `toml:"background"`
	Surface    string `toml:"surface"`
	Primary    string `toml:"primary"`
	Success    string `toml:"success"`
	Warning    string `toml:"warning"`
	Error      string `toml:"error"`
	Mailbox    string `toml:"mailbox"`
	Text       string `toml:"text"`
	TextDim    string `toml:"text_dim"`
	Border     string `toml:"border"`
}

func DefaultTheme() Theme {
	return Theme{
		Name:       "dark",
		Background: "232",
		Surface:    "235",
		Primary:    "39",
		Success:    "76",
		Warning:    "214",
		Error:      "196",
		Mailbox:    "140",
		Text:       "255",
		TextDim:    "245",
		Border:     "240",
	}
}

type ProfileStyle struct {
	Color string `toml:"color"`
	Style string `toml:"style"`
}

func DefaultProfileStyle() ProfileStyle {
	return ProfileStyle{}
}

type Config struct {
	DataDir        string                 `toml:"data_dir"`
	ListenAddrs    []string               `toml:"listen_addrs"`
	BootstrapPeers []string               `toml:"bootstrap_peers"`
	DHTEnabled     bool                   `toml:"dht_enabled"`
	Nickname       string                 `toml:"nickname"`
	ProfileStyle   ProfileStyle           `toml:"profile_style"`
	Relays         map[string]RelayConfig `toml:"relays"`
	Theme          Theme                  `toml:"theme"`
}

func DefaultConfig() *Config {
	return &Config{
		DataDir:        "~/.alkalyne",
		ListenAddrs:    []string{"/ip4/0.0.0.0/tcp/9000", "/ip4/0.0.0.0/udp/9000/quic-v1"},
		BootstrapPeers: []string{},
		DHTEnabled:     false,
		Nickname:       "",
		ProfileStyle:   DefaultProfileStyle(),
		Relays:         map[string]RelayConfig{},
		Theme:          DefaultTheme(),
	}
}
