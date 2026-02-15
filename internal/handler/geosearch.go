package handler

import (
	"strconv"

	"github.com/minhvip08/simis/internal/connection"
	"github.com/minhvip08/simis/internal/geo"
	"github.com/minhvip08/simis/internal/store"
	"github.com/minhvip08/simis/internal/utils"
)

type GeoSearchHandler struct{}
type geoSearchParams struct {
	key            string
	longitude      float64
	latitude       float64
	radiusInMeters float64
}

func parseGeoSearch(args []string) (*geoSearchParams, error) {
	if len(args) < 7 {
		return nil, ErrInvalidArguments
	}

	key := args[0]
	lon, lat, radius, radiusUnit := args[2], args[3], args[5], args[6]

	radiusInMeters, err := strconv.ParseFloat(radius, 64)
	if err != nil {
		return nil, err
	}

	if radiusUnit == "km" {
		radiusInMeters *= 1000.0
	}

	longitude, err := strconv.ParseFloat(lon, 64)
	if err != nil {
		return nil, err
	}

	latitude, err := strconv.ParseFloat(lat, 64)
	if err != nil {
		return nil, err
	}

	return &geoSearchParams{
		key:            key,
		longitude:      longitude,
		latitude:       latitude,
		radiusInMeters: radiusInMeters,
	}, nil
}

func executeGeoSearch(params *geoSearchParams) *ExecutionResult {
	sortedSet, ok := store.GetInstance().GetSortedSet(params.key)
	result := NewExecutionResult()
	if !ok {
		result.Response = utils.ToNullBulkString()
		return result
	}

	ranges := geo.GetSearchRanges(params.longitude, params.latitude, params.radiusInMeters)
	uniqMembers := make(map[string]bool)
	members := make([]string, 0)

	for _, r := range ranges {
		candidateMembers := sortedSet.GetRangeByValue(float64(r.Min), float64(r.Max))
		for _, member := range candidateMembers {
			lat, lon := geo.DecodeCoordinates(int64(member.Value))
			if !uniqMembers[member.Key] && geo.GetHaversineDistance(lat, lon, params.latitude, params.longitude) <= params.radiusInMeters {
				members = append(members, member.Key)
				uniqMembers[member.Key] = true
			}
		}
	}

	result.Response = utils.ToRespArray(members)
	return result
}

func (h *GeoSearchHandler) Execute(cmd *connection.Command) *ExecutionResult {
	params, err := parseGeoSearch(cmd.Args)
	if err != nil {
		return NewErrorResult(err)
	}
	return executeGeoSearch(params)
}
