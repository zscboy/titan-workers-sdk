package config

import (
	"github.com/BurntSushi/toml"
)

// Config represents the structure of the TOML file
type Config struct {
	Server   Server    `toml:"server"`
	Projects []Project `toml:"projects"`
	Socks5   Socks5    `toml:"socks5"`
	Http     Http      `toml:"http"`
	Tun      Tun       `toml:"tun"`
	Log      Log       `toml:""`
}

type Server struct {
	UserName string `toml:"user_name"`
	Password string `toml:"password"`
	URL      string `toml:"url"`
}

type Project struct {
	// ID string `toml:"id"`
	Region string `toml:"region"`
}

type Socks5 struct {
	ListenAddress string `toml:"listenAddress"`
}

type Http struct {
	ListenAddress string `toml:"listenAddress"`
}

type Tun struct {
	Count int `toml:"count"`
	Cap   int `toml:"cap"`
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
