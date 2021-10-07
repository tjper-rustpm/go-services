package graph

import (
	"errors"

	"github.com/google/uuid"
)

// This file is an extension of schema.resolvers.go and is not overwritten when
// gqlgen is used to generate resolvers.

// --- errors ---

var (
	errInternalServer = errors.New("internal server error; please contact support")
	errInvalidUUID    = errors.New("invalid UUID; please contact support")
)

func toUUIDs(ids []string) ([]uuid.UUID, error) {
	uuids := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		uuid, err := uuid.Parse(id)
		if err != nil {
			return nil, err
		}
		uuids = append(uuids, uuid)
	}
	return uuids, nil
}
