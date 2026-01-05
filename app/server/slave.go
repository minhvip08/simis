package server

import (
	"os"

	"github.com/minhvip08/simis/app/config"
)

type SlaveServer struct {
	config *config.Config
}

func NewSlaveServer(cfg *config.Config) *SlaveServer {
	return &SlaveServer{
		config: cfg,
	}
}

func (s *SlaveServer) Start() error {
	// TODO: Implement slave server start logic
	return nil
}

func (s *SlaveServer) Run() {
	if err := s.Start(); err != nil {
		os.Exit(1)
	}
}
