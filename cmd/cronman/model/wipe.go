package model

import (
	"database/sql"
	"sort"

	"github.com/tjper/rustcron/internal/model"

	"github.com/google/uuid"
)

type Wipes []Wipe

func (ws Wipes) CurrentWipe() Wipe {
	if len(ws) == 0 {
		return Wipe{}
	}

	ws.OrderCreatedDesc()
	return ws[0]
}

func (ws Wipes) OrderCreatedDesc() {
	sort.Slice(ws, func(i, j int) bool { return ws[i].CreatedAt.After(ws[j].CreatedAt) })
}

func (ws Wipes) Clone() Wipes {
	cloned := make(Wipes, 0, len(ws))
	for _, w := range ws {
		cloned = append(cloned, w.Clone())
	}
	return cloned
}

func (ws Wipes) Scrub() {
	for i := range ws {
		ws[i].Scrub()
	}
}

func NewMapWipe(seed, salt uint32) *Wipe {
	return &Wipe{
		Kind:    WipeKindMap,
		MapSeed: seed,
		MapSalt: salt,
	}
}

func NewFullWipe(seed, salt uint32) *Wipe {
	return &Wipe{
		Kind:    WipeKindFull,
		MapSeed: seed,
		MapSalt: salt,
	}
}

type Wipe struct {
	model.Model

	Kind     WipeKind
	MapSeed  uint32
	MapSalt  uint32
	ServerID uuid.UUID

	AppliedAt sql.NullTime
}

func (w Wipe) Clone() Wipe { return w }

func (w *Wipe) Scrub() {
	w.Model.Scrub()
	w.ServerID = uuid.Nil
}

type WipeKind string

const (
	WipeKindMap  WipeKind = "map"
	WipeKindFull WipeKind = "full"
)
