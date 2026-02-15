package handler

import (
	"fmt"
	"strconv"

	"github.com/minhvip08/simis/internal/connection"
	ds "github.com/minhvip08/simis/internal/datastructure"
	"github.com/minhvip08/simis/internal/geo"
	"github.com/minhvip08/simis/internal/store"
	"github.com/minhvip08/simis/internal/utils"
)

type GeoAddHandler struct{}

type geoAddParams struct {
	key       string
	locations []ds.KeyValue
}

func parseGeoAdd(args []string) (*geoAddParams, error) {
	if len(args) < 4 {
		return nil, ErrInvalidArguments
	}
	key := args[0]
	coordArgs := args[1:]
	if len(coordArgs)%3 != 0 {
		return nil, ErrInvalidArguments
	}

	locations := make([]ds.KeyValue, len(coordArgs)/3)
	for i := 0; i < len(coordArgs); i += 3 {
		lon, err := strconv.ParseFloat(coordArgs[i], 64)
		if err != nil {
			return nil, err
		}
		lat, err := strconv.ParseFloat(coordArgs[i+1], 64)
		if err != nil {
			return nil, err
		}

		if lat < geo.LatMin || lat > geo.LatMax || lon < geo.LonMin || lon > geo.LonMax {
			return nil, fmt.Errorf("invalid longitude,latitude pair %f,%f", lon, lat)
		}

		geo, err := geo.EncodeCoordinates(lat, lon)
		if err != nil {
			return nil, err
		}

		locations[i/3] = ds.KeyValue{
			Key:   coordArgs[i+2],
			Value: float64(geo),
		}
	}

	return &geoAddParams{key: key, locations: locations}, nil
}

func executeGeoAdd(params *geoAddParams) *ExecutionResult {
	db := store.GetInstance()
	sortedSet, _ := db.LoadOrStoreSortedSet(params.key)
	numAddedValues := sortedSet.Add(params.locations)
	result := NewExecutionResult()
	result.Response = utils.ToRespInt(numAddedValues)
	return result
}

func (h *GeoAddHandler) Execute(cmd *connection.Command) *ExecutionResult {
	params, err := parseGeoAdd(cmd.Args)
	if err != nil {
		return NewErrorResult(err)
	}
	return executeGeoAdd(params)
}
