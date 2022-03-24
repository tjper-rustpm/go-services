package hash

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFromStruct(t *testing.T) {
	tests := map[string]struct {
		src  interface{}
		hash map[string]interface{}
	}{
		"struct": {
			src: &struct {
				Name  string
				Value string
			}{Name: "a name", Value: "a value"},
			hash: map[string]interface{}{"Name": "a name", "Value": "a value"},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			actual, err := FromStruct(test.src)
			require.Nil(t, err)
			require.Equal(t, test.hash, actual)
		})
	}
}

func TestToStruct(t *testing.T) {
	type expected struct {
		dst interface{}
	}
	tests := map[string]struct {
		dst interface{}
		src map[string]interface{}
		exp expected
	}{
		"map": {
			dst: &struct {
				Name  string
				Value string
			}{},
			src: map[string]interface{}{"Name": "a name", "Value": "a value"},
			exp: expected{
				dst: &struct {
					Name  string
					Value string
				}{Name: "a name", Value: "a value"},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			err := ToStruct(test.dst, test.src)
			require.Nil(t, err)
			require.Equal(t, test.exp.dst, test.dst)
		})
	}
}
