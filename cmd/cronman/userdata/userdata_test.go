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
		ip           string
		identity     string
		hostName     string
		rconPassword string
		maxPlayers   int
		worldSize    int
		seed         int
		salt         int
		tickRate     int
		description  string
		opts         []Option
	}{
		"base": {
			ip:           "east-main.rustpm.com",
			identity:     "Rustpm East Main",
			hostName:     "rustpm-east-1",
			rconPassword: "rustpm-rconpassword",
			maxPlayers:   100,
			worldSize:    2000,
			seed:         123,
			salt:         321,
			tickRate:     30,
			description:  "Rustpm US East Main | Test Description",
			opts:         []Option{},
		},
		"mapwipe": {
			ip:           "east-main.rustpm.com",
			identity:     "Rustpm East Main",
			hostName:     "rustpm-east-1",
			rconPassword: "rustpm-rconpassword",
			maxPlayers:   100,
			worldSize:    2000,
			seed:         123,
			salt:         321,
			tickRate:     30,
			description:  "Rustpm US East Main | Test Description",
			opts:         []Option{WithMapWipe("Rustpm East Main")},
		},
		"blueprintwipe": {
			ip:           "east-main.rustpm.com",
			identity:     "Rustpm East Main",
			hostName:     "rustpm-east-1",
			rconPassword: "rustpm-rconpassword",
			maxPlayers:   100,
			worldSize:    2000,
			seed:         123,
			salt:         321,
			tickRate:     30,
			description:  "Rustpm US East Main | Test Description",
			opts:         []Option{WithBluePrintWipe("Rustpm East Main")},
		},
		"fullwipe": {
			ip:           "east-main.rustpm.com",
			identity:     "Rustpm East Main",
			hostName:     "rustpm-east-1",
			rconPassword: "rustpm-rconpassword",
			maxPlayers:   100,
			worldSize:    2000,
			seed:         123,
			salt:         321,
			tickRate:     30,
			description:  "Rustpm US East Main | Test Description",
			opts: []Option{
				WithBluePrintWipe("Rustpm East Main"),
				WithMapWipe("Rustpm East Main"),
			},
		},
		"queuebypass": {
			ip:           "east-main.rustpm.com",
			identity:     "Rustpm East Main",
			hostName:     "rustpm-east-1",
			rconPassword: "rustpm-rconpassword",
			maxPlayers:   100,
			worldSize:    2000,
			seed:         123,
			salt:         321,
			tickRate:     30,
			description:  "Rustpm US East Main | Test Description",
			opts:         []Option{WithQueueBypassPlugin()},
		},
		"usercfg": {
			ip:           "east-main.rustpm.com",
			identity:     "Rustpm East Main",
			hostName:     "rustpm-east-1",
			rconPassword: "rustpm-rconpassword",
			maxPlayers:   100,
			worldSize:    2000,
			seed:         123,
			salt:         321,
			tickRate:     30,
			description:  "Rustpm US East Main | Test Description",
			opts:         []Option{WithUserCfg("Rustpm East Main", []string{"user1", "user2", "user3"})},
		},
		"servercfg": {
			ip:           "east-main.rustpm.com",
			identity:     "Rustpm East Main",
			hostName:     "rustpm-east-1",
			rconPassword: "rustpm-rconpassword",
			maxPlayers:   100,
			worldSize:    2000,
			seed:         123,
			salt:         321,
			tickRate:     30,
			description:  "Rustpm US East Main | Test Description",
			opts:         []Option{WithServerCfg("Rustpm East Main", []string{"user1", "user2", "user3"})},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			userdata := Generate(
				test.identity,
				test.hostName,
				test.rconPassword,
				test.maxPlayers,
				test.worldSize,
				test.seed,
				test.salt,
				test.tickRate,
				test.description,
				test.opts...,
			)

			actual := []byte(userdata)
			if *golden {
				if err := ioutil.WriteFile(
					fmt.Sprintf("testdata/%s.golden", t.Name()),
					actual,
					0600,
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
