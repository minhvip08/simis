package command

import (
	"bufio"
	"bytes"
	"fmt"
	"strings"

	"github.com/minhvip08/simis/internal/utils"
)

type ParseCommand struct {
	Name string
	Args []string
}

func ParseRequest(data string) (string, []string, error) {
	parsedArray, err := utils.ParseRESPArray(data)
	if err != nil {
		return "", nil, fmt.Errorf("error parsing command: %w", err)
	}

	// Flatten nested arrays to []string for command processing
	commands := utils.FlattenRESPArray(parsedArray)
	if len(commands) == 0 {
		return "", nil, fmt.Errorf("empty command")
	}

	command := strings.ToUpper(strings.TrimSpace(commands[0]))
	args := commands[1:]

	return command, args, nil
}

func ParseMultipleRequests(data string) ([]ParseCommand, string, error) {
	if len(data) == 0 {
		return nil, "", nil
	}

	reader := bufio.NewReader(bytes.NewBufferString(data))
	commands := make([]ParseCommand, 0)

	// Iteratively parse multiple RESP arrays from the data
	for {
		_, err := reader.Peek(1)

		// No more data to read
		if err != nil {
			break
		}

		parsedArray, err := utils.ParseRESPArrayRecursive(reader)
		if err != nil {
			var remaining bytes.Buffer
			_, _ = remaining.ReadFrom(reader)
			return commands, remaining.String(), nil
		}

		// Convert parsed array to command and args
		flatCommands := utils.FlattenRESPArray(parsedArray)
		if len(flatCommands) > 0 {
			cmdName := strings.ToUpper(strings.TrimSpace(flatCommands[0]))
			cmdArgs := flatCommands[1:]
			commands = append(commands, ParseCommand{
				Name: cmdName,
				Args: cmdArgs,
			})
		}

	}
	var leftover bytes.Buffer
	_, _ = leftover.ReadFrom(reader)
	return commands, leftover.String(), nil
}
