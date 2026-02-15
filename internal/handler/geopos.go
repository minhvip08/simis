package handler

import (
	"strconv"

	"github.com/minhvip08/simis/internal/connection"
	"github.com/minhvip08/simis/internal/geo"
	"github.com/minhvip08/simis/internal/store"
	"github.com/minhvip08/simis/internal/utils"
)

type GeoPosHandler struct{}

type geoPosParams struct {
	key     string
	members []string
}

func parseGeoPos(args []string) (*geoPosParams, error) {
	if len(args) < 2 {
		return nil, ErrInvalidArguments
	}
	return &geoPosParams{
		key:     args[0],
		members: args[1:],
	}, nil
}

func executeGeoPos(params *geoPosParams) *ExecutionResult {
	db := store.GetInstance()
	result := NewExecutionResult()

	sortedSet, ok := db.GetSortedSet(params.key)
	coordinates := make([]string, len(params.members))

	if !ok {
		for i := range coordinates {
			coordinates[i] = utils.ToRespArray(nil)
		}
		result.Response = utils.ToSimpleRespArray(coordinates)
		return result
	}

	for i, member := range params.members {
		score, found := sortedSet.GetScore(member)
		if !found {
			coordinates[i] = utils.ToRespArray(nil)
		} else {
			lat, lon := geo.DecodeCoordinates(int64(score))
			coordinates[i] = utils.ToRespArray([]string{
				strconv.FormatFloat(lon, 'f', -1, 64),
				strconv.FormatFloat(lat, 'f', -1, 64),
			})

		}
	}
	result.Response = utils.ToSimpleRespArray(coordinates)
	return result
}

func (h *GeoPosHandler) Execute(cmd *connection.Command) *ExecutionResult {
	params, err := parseGeoPos(cmd.Args)
	if err != nil {
		return NewErrorResult(err)
	}
	return executeGeoPos(params)
}
