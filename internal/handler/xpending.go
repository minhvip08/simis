package handler

import (
	"strconv"
	"strings"
	"time"

	"github.com/minhvip08/simis/internal/connection"
	"github.com/minhvip08/simis/internal/store"
	"github.com/minhvip08/simis/internal/utils"
)

type XPendingHandler struct {
}

func (h *XPendingHandler) Execute(cmd *connection.Command) *ExecutionResult {
	result := NewExecutionResult()

	// XPENDING key group [start end count] [consumer]
	if len(cmd.Args) < 2 {
		result.Error = ErrInvalidArguments
		return result
	}

	streamKey := cmd.Args[0]
	groupName := cmd.Args[1]

	db := store.GetInstance()
	stream, ok := db.GetStream(streamKey)
	if !ok {
		result.Error = ErrInvalidArguments
		return result
	}

	// Get the consumer group
	group, err := stream.GetGroup(groupName)
	if err != nil {
		result.Error = err
		return result
	}

	// If no additional args, return summary
	if len(cmd.Args) == 2 {
		// Return [count, start_id, end_id, [[consumer, count], ...]]
		var builder strings.Builder
		builder.WriteString("*4\r\n")

		// Count of pending entries
		builder.WriteString(utils.ToRespInt(len(group.PendingEntries)))

		// Start ID of pending range
		if len(group.PendingEntries) > 0 {
			minID := group.LastDeliveredID
			for _, pending := range group.PendingEntries {
				if pending.ID.Compare(minID) < 0 {
					minID = pending.ID
				}
			}
			builder.WriteString(utils.ToBulkString(minID.String()))
		} else {
			builder.WriteString("$-1\r\n")
		}

		// End ID of pending range
		if len(group.PendingEntries) > 0 {
			maxID := group.LastDeliveredID
			for _, pending := range group.PendingEntries {
				if pending.ID.Compare(maxID) > 0 {
					maxID = pending.ID
				}
			}
			builder.WriteString(utils.ToBulkString(maxID.String()))
		} else {
			builder.WriteString("$-1\r\n")
		}

		// Consumer list
		builder.WriteString("*")
		builder.WriteString(strconv.Itoa(len(group.Consumers)))
		builder.WriteString("\r\n")
		for consumerName, consumer := range group.Consumers {
			builder.WriteString("*2\r\n")
			builder.WriteString(utils.ToBulkString(consumerName))
			builder.WriteString(utils.ToRespInt(consumer.PendingEntriesNum))
		}

		result.Response = builder.String()
		return result
	}

	// If args provided, return detailed pending entries
	consumerFilter := ""
	if len(cmd.Args) > 5 {
		consumerFilter = cmd.Args[5]
	}

	// Get pending entries
	pendingEntries, err := stream.GetPendingEntries(groupName, consumerFilter)
	if err != nil {
		result.Error = err
		return result
	}

	// Format response as array of [id, consumer, idle_time, delivery_count]
	var builder strings.Builder
	builder.WriteString("*")
	builder.WriteString(strconv.Itoa(len(pendingEntries)))
	builder.WriteString("\r\n")

	for _, pending := range pendingEntries {
		builder.WriteString("*4\r\n")
		// Entry ID
		builder.WriteString(utils.ToBulkString(pending.ID.String()))
		// Consumer name
		if pending.Consumer == "" {
			builder.WriteString("$-1\r\n")
		} else {
			builder.WriteString(utils.ToBulkString(pending.Consumer))
		}
		// Idle time in milliseconds
		now := time.Now().UnixMilli()
		idleTime := now - pending.DeliveredTime
		builder.WriteString(utils.ToRespInt(int(idleTime)))
		// Delivery count
		builder.WriteString(utils.ToRespInt(pending.DeliveryCount))
	}

	result.Response = builder.String()
	return result
}
