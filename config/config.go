package config

import (
	"github.com/BurntSushi/toml"
)

// Config represents the structure of the TOML file
type Config struct {
	Server          Server          `toml:"server"`
	Node            Node            `toml:"node"`
	Socks5          Socks5          `toml:"socks5"`
	Http            Http            `toml:"http"`
	LocalHttpServer LocalHttpServer `toml:"local_http_server"`
	Tun             Tun             `toml:"tun"`
	Log             Log             `toml:"log"`
	Selector        Selector        `toml:"selector"`
}

type Server struct {
	UserName string `toml:"user_name"`
	Password string `toml:"password"`
	URL      string `toml:"url"`
}

type Node struct {
	ID string `toml:"id"`
	// Region string `toml:"region"`
}

// type IP struct {
// 	Type string `toml:"type"`
// }

type Selector struct {
	Type          string `toml:"type"`
	AreaID        string `toml:"area_id"`
	DefaultNodeID string `toml:"default_node_id"`
}

type Socks5 struct {
	ListenAddress string `toml:"listenAddress"`
}

type Http struct {
	ListenAddress string `toml:"listenAddress"`
}

type LocalHttpServer struct {
	ListenAddress string `toml:"listenAddress"`
}

type Tun struct {
	Count int `toml:"count"`
	Cap   int `toml:"cap"`

	URL     string `toml:"url"`
	AuthKey string `toml:"authKey"`
}

type Log struct {
	Level string `toml:"level"`
}

func ParseConfig(filePath string) (*Config, error) {
	var config Config

	// Read and decode the TOML file
	if _, err := toml.DecodeFile(filePath, &config); err != nil {
		return nil, err
	}
	return &config, nil
}
