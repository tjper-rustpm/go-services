package model

import "time"

func (r LoginUserResult) Clone() LoginUserResult {
	user := r.User.Clone()
	r.User = &user
	return r
}

func (r *LoginUserResult) Scrub() {
	r.User.Scrub()
}

func (u User) Clone() User {
	return u
}

func (u *User) Scrub() {
	u.ID = ""
	u.VerifiedAt = nil
	u.UpdatedAt = time.Time{}
	u.CreatedAt = time.Time{}
}
