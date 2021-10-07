package model

// RoleKind represent the various roles a user can have.
type RoleKind string

const (
	// RoleKindStandard is the default role, and the most limited.
	RoleKindStandard RoleKind = "STANDARD"

	// RoleKindModerator is a role with permissions that allow for custodial
	// tasks to be done.
	RoleKindModerator RoleKind = "MODERATOR"

	// RoleKindAdmin is a role with elevated permissions that allows for any
	// interaction.
	RoleKindAdmin RoleKind = "ADMIN"
)
