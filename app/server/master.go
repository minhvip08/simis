package server

import (
	"net"
	"os"
	"strconv"

	"github.com/minhvip08/simis/app/config"
	"github.com/minhvip08/simis/app/connection"
	"github.com/minhvip08/simis/app/logger"
)

type MasterServer struct {
	config *config.Config
}

func NewMasterServer(cfg *config.Config) *MasterServer {
	return &MasterServer{
		config: cfg,
	}
}

func (m *MasterServer) Start() error {

	localAddr := net.JoinHostPort(m.config.Host, strconv.Itoa(m.config.Port))
	listener, err := net.Listen("tcp", localAddr)
	if err != nil {
		logger.Error("Failed to start master server: %v", err)
		return err
	}
	defer listener.Close()
	logger.Info("Master server listening on %s", localAddr)

	for {
		conn, err := listener.Accept()
		if err != nil {
			logger.Error("Error accepting connection", "error", err)
			continue
		}

		logger.Info("Accepted connection from %s", conn.RemoteAddr().String())
		go Handle(connection.NewRedisConnection(conn))

	}

}

func (m *MasterServer) Run() {
	if err := m.Start(); err != nil {
		os.Exit(1)
	}
}
