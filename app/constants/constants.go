package constants

const (
	DefaultBufferSize       = 4096
	SmallBufferSize         = 1024
	DefaultCommandQueueSize = 1000
	DefaultTransactionQueueSize = 1000
)

type CommandType string

const (
	CommandPing CommandType = "PING"
)

var PubSubCommands = map[string]bool{
	string(CommandPing): true,
}