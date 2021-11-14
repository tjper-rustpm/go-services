package model

import (
	"github.com/google/uuid"
)

type LiveServers []LiveServer

func (s LiveServers) Clone() LiveServers {
	cloned := make(LiveServers, 0, len(s))
	cloned = append(cloned, s...)
	return cloned
}

func (s LiveServers) Scrub() {
	for i := range s {
		s[i].Scrub()
	}
}

// LiveServer is a server that is accessible to players and is scheduled to
// stop at some point in the future.
type LiveServer struct {
	Model

	ServerDefinitionID uuid.UUID
	ServerDefinition   ServerDefinition

	AssociationID string
	ActivePlayers uint8
	QueuedPlayers uint8
}

func (s LiveServer) Clone() LiveServer {
	return s
}

func (s *LiveServer) Scrub() {
	s.Model.Scrub()
	s.ServerDefinitionID = uuid.Nil
	s.AssociationID = ""
	s.ServerDefinition.Scrub()
}

type DormantServers []DormantServer

func (s DormantServers) Clone() DormantServers {
	cloned := make(DormantServers, 0, len(s))
	cloned = append(cloned, s...)
	return cloned
}

func (s DormantServers) Scrub() {
	for i := range s {
		s[i].Scrub()
	}
}

// DormantServer is a server that is not accessible to players and is scheduled
// to start at some point in the future.
type DormantServer struct {
	Model

	ServerDefinitionID uuid.UUID
	ServerDefinition   ServerDefinition
}

func (s DormantServer) Clone() DormantServer {
	return s
}

func (s *DormantServer) Scrub() {
	s.Model.Scrub()
	s.ServerDefinitionID = uuid.Nil
	s.ServerDefinition.Scrub()
}

type ArchivedServers []ArchivedServer

func (s ArchivedServers) Clone() ArchivedServers {
	cloned := make(ArchivedServers, 0, len(s))
	cloned = append(cloned, s...)
	return cloned
}

func (s ArchivedServers) Scrub() {
	for i := range s {
		s[i].Scrub()
	}
}

// ArchivedServer is a server that is not accessible to players and is
// not scheduled to start at any point in the future.
type ArchivedServer struct {
	Model

	ServerDefinitionID uuid.UUID
	ServerDefinition   ServerDefinition
}

func (s ArchivedServer) Clone() ArchivedServer {
	return s
}

func (s *ArchivedServer) Scrub() {
	s.Model.Scrub()
	s.ServerDefinitionID = uuid.Nil
	s.ServerDefinition.Scrub()
}
