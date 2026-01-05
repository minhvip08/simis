package main

import (
	"flag"
	"net"
	"os"

	"github.com/minhvip08/simis/app/command"
	"github.com/minhvip08/simis/app/config"
	"github.com/minhvip08/simis/app/executor"
	"github.com/minhvip08/simis/app/server"
)

// Ensures gofmt doesn't remove the "net" and "os" imports in stage 1 (feel free to remove this!)
var _ = net.Listen
var _ = os.Exit

func parseFlags() (int, config.ServerRole, string, string, string) {
	var port int
	var masterAddress string
	var dir string
	var fileName string
	// TODO: add a flag for role (master/slave)

	flag.IntVar(&port, "port", 6379, "The port to listen on")
	flag.StringVar(&masterAddress, "replicaof", "", "The address of the master server (for slaves)")
	flag.StringVar(&dir, "dir", "/tmp/redis-data", "The working directory")
	flag.StringVar(&fileName, "fileName", "temp.rdb", "The name of the RDB file")
	flag.Parse()

	role := config.Master
	if masterAddress != "" {
		role = config.Slave
	}
	return port, role, masterAddress, dir, fileName
}

func main() {

	port, role, masterAddress, dir, fileName := parseFlags()
	config.Init(&config.ConfigOptions{
		Port:          port,
		Role:          role,
		MasterAddress: masterAddress,
		Dir:           dir,
		FileName:      fileName,
	})

	// TODO: Load RDB file

	command.GetQueueInstance()

	exec := executor.NewExecutor()
	go exec.Start() // Start the executor in a separate goroutine

	server := server.NewServer(config.GetInstance())
	server.Run()
}
