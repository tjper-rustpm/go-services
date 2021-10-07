package admin

func NewAdminSet(admins []string) *AdminSet {
	return &AdminSet{
		admins: admins,
	}
}

type AdminSet struct {
	admins []string
}

func (s AdminSet) Contains(admin string) bool {
	for i := range s.admins {
		if s.admins[i] == admin {
			return true
		}
	}
	return false
}
