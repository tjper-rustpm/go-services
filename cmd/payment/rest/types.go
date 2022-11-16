package rest

import (
	"time"

	"github.com/google/uuid"
	"github.com/tjper/rustcron/cmd/payment/model"
)

// Vip is a very-important-person.
type Vip struct {
	// ID is the Vip's unique identifier.
	ID uuid.UUID `json:"id"`
	// ServerID is the ID of server the VIP is tied to.
	ServerID uuid.UUID `json:"serverId"`
	// Status is the current status of the VIP. See the type for
	// possible values and their explanation.
	Status VipStatus `json:"status"`
	// CreatedAt is when the VIP was created.
	CreatedAt time.Time `json:"createdAt"`
}

// Vips is a collection of Vip instances.
type Vips []Vip

// FromModelVips converts model.Vips into Vips.
func (vips *Vips) FromModelVips(froms model.Vips) {
	if vips == nil {
		*vips = make(Vips, 0, len(froms))
	}
	for _, from := range froms {
    status := Active
    if time.Now().After(from.ExpiresAt) {
      status = Expired
    }

		*vips = append(
			*vips,
			Vip{
				ID:        from.ID,
				ServerID:  from.ServerID,
				Status:    status,
				CreatedAt: from.CreatedAt,
			},
		)
	}
}

// VipStatus is a type encompassing the range of possible VIP statuses.
type VipStatus string

const (
  // Active is a VIP that is current receiving the benefits of being a VIP.
  Active VipStatus = "active"
  // Expired is a VIP that is no longer active.
  Expired VipStatus = "expired"
)

// Redirect contains a URL that should be redirected to by the client. This is
// used instead of http.Redirect as not all client software is capable of
// following a http.Redirect.
type Redirect struct {
	URL string `json:"url"`
}
