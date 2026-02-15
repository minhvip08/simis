package handler

import (
	"strconv"

	"github.com/minhvip08/simis/internal/connection"
	"github.com/minhvip08/simis/internal/geo"
	"github.com/minhvip08/simis/internal/store"
	"github.com/minhvip08/simis/internal/utils"
)

type GeoDistHandler struct{}

type geoDistParams struct {
	key string
	p1  string
	p2  string
}

func parseGeoDist(args []string) (*geoDistParams, error) {
	if len(args) < 3 {
		return nil, ErrInvalidArguments
	}
	return &geoDistParams{
		key: args[0],
		p1:  args[1],
		p2:  args[2],
	}, nil
}

func executeGeoDist(params *geoDistParams) *ExecutionResult {
	sortedSet, ok := store.GetInstance().GetSortedSet(params.key)
	result := NewExecutionResult()
	if !ok {
		result.Response = utils.ToNullBulkString()
		return result
	}

	score1, found1 := sortedSet.GetScore(params.p1)
	score2, found2 := sortedSet.GetScore(params.p2)

	if !found1 || !found2 {
		result.Response = utils.ToNullBulkString()
		return result
	}

	lat1, lon1 := geo.DecodeCoordinates(int64(score1))
	lat2, lon2 := geo.DecodeCoordinates(int64(score2))

	distance := geo.GetHaversineDistance(lat1, lon1, lat2, lon2)
	result.Response = utils.ToBulkString(strconv.FormatFloat(distance, 'f', -1, 64))
	return result
}

func (h *GeoDistHandler) Execute(cmd *connection.Command) *ExecutionResult {
	params, err := parseGeoDist(cmd.Args)
	if err != nil {
		return NewErrorResult(err)
	}
	return executeGeoDist(params)
}
