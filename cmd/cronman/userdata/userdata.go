package userdata

import (
	"fmt"
	"strings"
)

const (
	launchTemplate = `
export LD_LIBRARY_PATH=/home/rustserver:/home/rustserver/RustDedicated:{LD_LIBRARY_PATH};

echo "--- Starting Dedicated Server\n"
while true; do
  su -c "/home/rustserver/RustDedicated -batchmode -nographics \
    -server.ip \"0.0.0.0\" \
    -server.identity \"rustpm\" \
    -server.hostname \"%s\" \
    -server.port \"28015\" \
    -rcon.ip \"0.0.0.0\" \
    -rcon.password \"%s\" \
    -rcon.web \"1\" \
    -rcon.port \"28016\" \
    -app.listenip \"0.0.0.0\" \
    -app.port \"28082\" \
    -server.maxplayers %d \
    -server.worldsize %d \
    -server.seed %d \
    -server.salt %d \
    -server.tickrate %d \
    -server.saveinterval 300 \
    -logfile" - rustserver
  echo "\n--- Restarting Dedicated Server\n"
done
--//
`

	userCfgTemplate = `
su -c "cat <<EOT > /home/rustserver/server/rustpm/cfg/users.cfg
%s
EOT" -  rustserver
`
	serverCfgTemplate = `
su -c "cat <<EOT > /home/rustserver/server/rustpm/cfg/server.cfg
oxide.group remove vip
oxide.group add vip
oxide.grant group vip bypassqueue.allow
%s
EOT" -  rustserver
`

	cfgDirectoryScript = `
su -c "mkdir -p /home/rustserver/server/rustpm/cfg" - rustserver
`

	installScript = `Content-Type: multipart/mixed; boundary="//"
MIME-Version: 1.0

--//
Content-Type: text/cloud-config; charset="us-ascii"
MIME-Version: 1.0
Content-Transfer-Encoding: 7bit
Content-Disposition: attachment; filename="cloud-config.txt"

#cloud-config
cloud_final_modules:
- [scripts-user, always]

--//
Content-Type: text/x-shellscript; charset="us-ascii"
MIME-Version: 1.0
Content-Transfer-Encoding: 7bit
Content-Disposition: attachment; filename="userdata.txt"

#!/bin/bash
dpkg --add-architecture i386
apt-get update && apt-get upgrade -y
apt-get install -y \
  ca-certificates \
  lib32gcc1 \
  libsdl2-2.0-0:i386 \
  libsdl2-2.0-0 \
  sqlite3 \
	docker.io \
	unzip


echo steamcmd steam/license note '' | debconf-set-selections
echo steamcmd steam/question select "I AGREE" | debconf-set-selections
apt-get install -y steamcmd
ln -s /usr/games/steamcmd /usr/bin/steamcmd

id -u rustserver &>/dev/null || adduser --disabled-password --gecos "" rustserver
su -c "steamcmd +login anonymous +force_install_dir /home/rustserver/ +app_update 258550 validate +quit" - rustserver
`

	bluePrintWipeScript = `
find /home/rustserver/server/rustpm -name "player\.blueprints\.*\.db" | xargs rm
 `
	mapWipeScript = `
find /home/rustserver/server/rustpm -name 'proceduralmap\.*\.*\.*\.map' | xargs rm
 `

	installOxideScript = `
su -c "curl --output Oxide.Rust-linux.zip -L https://github.com/OxideMod/Oxide.Rust/releases/latest/download/Oxide.Rust-linux.zip" - rustserver
su -c "unzip -o -d /home/rustserver/ Oxide.Rust-linux.zip" - rustserver
`

	installBypassQueuePluginScript = `
su -c "curl https://umod.org/plugins/BypassQueue.cs --output /home/rustserver/oxide/plugins/BypassQueue.cs --create-dirs" - rustserver
`
)

// Generate userdata to be used as an AWS EC2 instance's user data. Userdata is
// executed when an EC2 instance starts.
func Generate(
	hostName string,
	rconPassword string,
	maxPlayers int,
	worldSize int,
	seed int,
	salt int,
	tickRate int,
	opts ...Option,
) string {
	var s strings.Builder
	s.WriteString(installScript)
	s.WriteString(installOxideScript)
	s.WriteString(cfgDirectoryScript)
	for _, opt := range opts {
		s.WriteString(opt())
	}
	s.WriteString(
		fmt.Sprintf(
			launchTemplate,
			hostName,
			rconPassword,
			maxPlayers,
			worldSize,
			seed,
			salt,
			tickRate,
		))

	return s.String()
}

// Option is a userdata option that is used to configure the userdata.
// Typically, Option is passed to Generate.
type Option func() string

// WithBluePrintWipe returns an Option that enables the generation of a
// blueprint wipe script via Generate.
func WithBluePrintWipe() Option {
	return func() string {
		return bluePrintWipeScript
	}
}

// WithMapWipe returns an Option that enables the generation of a map wipe
// script via Generate.
func WithMapWipe() Option {
	return func() string {
		return mapWipeScript
	}
}

// WithQueueBypassPlugin returns an Option that enables the queue bypass oxide
// plugin.
func WithQueueBypassPlugin() Option {
	return func() string {
		return installBypassQueuePluginScript
	}
}

// WithUserCfg returns an Option that configures the userdata to create a user
// config.
func WithUserCfg(steamIDs []string) Option {
	cmds := make([]string, 0, len(steamIDs))
	for _, id := range steamIDs {
		cmds = append(cmds, fmt.Sprintf("moderatorid %s", id))
	}
	return func() string {
		return fmt.Sprintf(userCfgTemplate, strings.Join(cmds, "\n"))
	}
}

// WithServerCfg returns an Option that configures the userdata to create a
// server config.
func WithServerCfg(steamIDs []string) Option {
	cmds := make([]string, 0, len(steamIDs))
	for _, id := range steamIDs {
		cmds = append(cmds, fmt.Sprintf("oxide.usergroup add %s vip", id))
	}
	return func() string {
		return fmt.Sprintf(serverCfgTemplate, strings.Join(cmds, "\n"))
	}
}
