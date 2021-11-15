package rest

import (
	"encoding/json"
)

func jsonConversion(from, to interface{}) error {
	b, err := json.Marshal(from)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(b, to); err != nil {
		return err
	}
	return nil
}
