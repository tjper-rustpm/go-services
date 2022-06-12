package userdata

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"testing"
)

var golden = flag.Bool("golden", false, "enable golden tests to overwrite .golden files")

func TestGenerate(t *testing.T) {
	tests := map[string]struct {
		identity     string
		hostName     string
		rconPassword string
		maxPlayers   int
		worldSize    int
		seed         int
		salt         int
		tickRate     int
		opts         []Option
	}{
		"base": {
			hostName:     "rustpm-east-1",
			rconPassword: "rustpm-rconpassword",
			maxPlayers:   100,
			worldSize:    2000,
			seed:         123,
			salt:         321,
			tickRate:     30,
			opts:         []Option{},
		},
		"mapwipe": {
			hostName:     "rustpm-east-1",
			rconPassword: "rustpm-rconpassword",
			maxPlayers:   100,
			worldSize:    2000,
			seed:         123,
			salt:         321,
			tickRate:     30,
			opts:         []Option{WithMapWipe()},
		},
		"blueprintwipe": {
			hostName:     "rustpm-east-1",
			rconPassword: "rustpm-rconpassword",
			maxPlayers:   100,
			worldSize:    2000,
			seed:         123,
			salt:         321,
			tickRate:     30,
			opts:         []Option{WithBluePrintWipe()},
		},
		"fullwipe": {
			hostName:     "rustpm-east-1",
			rconPassword: "rustpm-rconpassword",
			maxPlayers:   100,
			worldSize:    2000,
			seed:         123,
			salt:         321,
			tickRate:     30,
			opts: []Option{
				WithBluePrintWipe(),
				WithMapWipe(),
			},
		},
		"queuebypass": {
			hostName:     "rustpm-east-1",
			rconPassword: "rustpm-rconpassword",
			maxPlayers:   100,
			worldSize:    2000,
			seed:         123,
			salt:         321,
			tickRate:     30,
			opts:         []Option{WithQueueBypassPlugin()},
		},
		"usercfg": {
			hostName:     "rustpm-east-1",
			rconPassword: "rustpm-rconpassword",
			maxPlayers:   100,
			worldSize:    2000,
			seed:         123,
			salt:         321,
			tickRate:     30,
			opts:         []Option{WithUserCfg([]string{"user1", "user2", "user3"})},
		},
		"servercfg": {
			hostName:     "rustpm-east-1",
			rconPassword: "rustpm-rconpassword",
			maxPlayers:   100,
			worldSize:    2000,
			seed:         123,
			salt:         321,
			tickRate:     30,
			opts:         []Option{WithServerCfg([]string{"user1", "user2", "user3"})},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			userdata := Generate(
				test.hostName,
				test.rconPassword,
				test.maxPlayers,
				test.worldSize,
				test.seed,
				test.salt,
				test.tickRate,
				test.opts...,
			)

			actual := []byte(userdata)
			if *golden {
				if err := ioutil.WriteFile(
					fmt.Sprintf("testdata/%s.golden", t.Name()),
					actual,
					0644,
				); err != nil {
					t.Error(err)
					return
				}
			}

			expected, err := ioutil.ReadFile(
				fmt.Sprintf("testdata/%s.golden", t.Name()),
			)
			if err != nil {
				t.Error(err)
				return
			}

			if !bytes.Equal(expected, actual) {
				t.Errorf(
					"unexpected userdata\nexpected: %s\nactual: %s\n",
					expected,
					actual,
				)
				return
			}
		})
	}
}
