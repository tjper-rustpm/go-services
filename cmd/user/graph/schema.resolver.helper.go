package graph

import (
	graphmodel "github.com/tjper/rustcron/cmd/user/graph/model"
	"github.com/tjper/rustcron/cmd/user/model"
)

// --- helpers ---

func toModelUser(user model.User) *graphmodel.User {
	modelUser := &graphmodel.User{
		ID:        user.ID.String(),
		Email:     user.Email,
		Role:      user.Role,
		UpdatedAt: user.UpdatedAt,
		CreatedAt: user.CreatedAt,
	}

	if user.VerifiedAt.Valid {
		modelUser.VerifiedAt = &user.VerifiedAt.Time
	}
	return modelUser
}
