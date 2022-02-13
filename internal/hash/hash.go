package hash

import (
	"encoding/json"
	"fmt"
)

func FromStruct(src interface{}) (map[string]interface{}, error) {
	b, err := json.Marshal(src)
	if err != nil {
		return nil, fmt.Errorf("hash; type: %T, error: %w", src, err)
	}

	m := make(map[string]interface{})
	if err = json.Unmarshal(b, &m); err != nil {
		return nil, fmt.Errorf("hash; type: %T, error: %w", src, err)
	}

	return m, nil
}

func ToStruct(dst interface{}, src map[string]interface{}) error {
	b, err := json.Marshal(src)
	if err != nil {
		return fmt.Errorf("from hash; type: %T, error: %w", src, err)
	}

	if err := json.Unmarshal(b, dst); err != nil {
		return fmt.Errorf("hash; type: %T, error: %w", src, err)
	}

	return nil
}
