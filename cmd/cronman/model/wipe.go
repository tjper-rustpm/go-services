package model

import (
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
	sort.Slice(ws, func(i, j int) bool { return ws[i].CreatedAt.Before(ws[j].CreatedAt) })
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

type Wipe struct {
	model.Model

	MapSeed  uint16
	MapSalt  uint16
	ServerID uuid.UUID
}

func (w Wipe) Clone() Wipe { return w }

func (w *Wipe) Scrub() {
	w.Model.Scrub()
	w.ServerID = uuid.Nil
}
