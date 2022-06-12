package admins

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestContains(t *testing.T) {
	type expected struct {
		contains bool
	}
	tests := map[string]struct {
		admins []string
		admin  string
		exp    expected
	}{
		"is admin": {
			admins: []string{
				"admin1@email.com",
				"admin2@email.com",
			},
			admin: "admin1@email.com",
			exp: expected{
				contains: true,
			},
		},
		"is not admin": {
			admins: []string{
				"admin1@email.com",
				"admin2@email.com",
			},
			admin: "admin3@email.com",
			exp: expected{
				contains: false,
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			admins := New(test.admins)
			require.Equal(t, test.exp.contains, admins.Contains(test.admin))
		})
	}
}
