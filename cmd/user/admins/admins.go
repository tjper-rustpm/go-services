package admins

func New(admins []string) Admins {
	return Admins(admins)
}

type Admins []string

func (s Admins) Contains(admin string) bool {
	for i := range s {
		if s[i] == admin {
			return true
		}
	}
	return false
}
