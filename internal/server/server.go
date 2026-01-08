package server

import "github.com/minhvip08/simis/internal/config"

type Server interface {
	Start() error
	Run()
}

func NewServer(cfg *config.Config) Server {
	if cfg.Role == config.Master {
		return NewMasterServer(cfg)
	} else {
		return NewSlaveServer(cfg)
	}
}
