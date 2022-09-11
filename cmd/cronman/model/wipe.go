package model

import (
	"math/rand"
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

func NewMapWipe(seed, salt uint16) *Wipe {
	return &Wipe{
		Kind:    WipeKindMap,
		MapSeed: seed,
		MapSalt: salt,
	}
}

func NewFullWipe(seed, salt uint16) *Wipe {
	return &Wipe{
		Kind:    WipeKindFull,
		MapSeed: seed,
		MapSalt: salt,
	}
}

type Wipe struct {
	model.Model

	Kind     WipeKind
	MapSeed  uint16
	MapSalt  uint16
	ServerID uuid.UUID
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

// GenerateSeed creates a new random seed that may be used as a wipe's map
// seed.
func GenerateSeed() uint16 {
	const max = 1000000
	return uint16(rand.Intn(max) + 1)
}

// GenerateSalt creates a new random salt that may be used as a wipe's map
// salt.
func GenerateSalt() uint16 {
	const max = 1000000
	return uint16(rand.Intn(max) + 1)
}
