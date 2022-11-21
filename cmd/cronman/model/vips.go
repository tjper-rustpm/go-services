package model

import (
	"time"

	"github.com/tjper/rustcron/internal/model"

	"github.com/google/uuid"
)

// Vips is a slice of Vip instances.
type Vips []Vip

// Active filters and returns retrieves the subset of active Vips.
func (vs Vips) Active() Vips {
	var vips Vips
	for _, vip := range vs {
		if !time.Now().Before(vip.ExpiresAt) {
			continue
		}
		vips = append(vips, vip)
	}
	return vips
}

// SteamIDs retrieves the Vips set of steam IDs.
func (vs Vips) SteamIDs() []string {
	var steamIDs []string
	for _, vip := range vs {
		steamIDs = append(steamIDs, vip.SteamID)
	}
	return steamIDs
}

// Equal checks if the Vips instance is equal to the passed Vips instance.
func (vs Vips) Equal(vs2 Vips) bool {
	if len(vs) != len(vs2) {
		return false
	}
	for i := range vs {
		if !vs[i].Equal(vs2[i]) {
			return false
		}
	}
	return true
}

func (vs Vips) Scrub() {
	for i := range vs {
		vs[i].Scrub()
	}
}

func (vs Vips) Clone() Vips {
	cloned := make(Vips, 0, len(vs))
	for _, vip := range vs {
		cloned = append(cloned, vip.Clone())
	}
	return cloned
}

// Vip is a "very important person" on a cronman server. The are granted
// special privileges such as queue skip.
type Vip struct {
	model.Model
	ServerID  uuid.UUID
	SteamID   string
	ExpiresAt time.Time
}

// Equal checks if the Vip instance is equal to the passed Vip instance.
func (vip Vip) Equal(vip2 Vip) bool {
	equal := true
	equal = equal && vip.Model.Equal(vip2.Model)
	equal = equal && vip.ExpiresAt.Equal(vip2.ExpiresAt)
	equal = equal && vip.ServerID == vip2.ServerID
	equal = equal && vip.SteamID == vip2.SteamID
	return equal
}

func (vip Vip) Clone() Vip {
	return vip
}

func (vip *Vip) Scrub() {
	vip.Model.Scrub()
	vip.ServerID = uuid.Nil
}
