package config

type ServerRole string

const (
	Master ServerRole = "master"
	Slave  ServerRole = "slave"	
)
