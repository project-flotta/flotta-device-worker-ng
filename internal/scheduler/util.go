package scheduler

type id interface {
	ID() string
}

// substract return all elements of a which are not found in b
func substract[S, T id](a []S, b []T) []S {
	m1 := make(map[string]S)
	m2 := make(map[string]T)

	limit := len(a)
	if limit < len(b) {
		limit = len(b)
	}

	for i := 0; i < limit; i++ {
		if i < len(a) {
			m1[a[i].ID()] = a[i]
		}

		if i < len(b) {
			m2[b[i].ID()] = b[i]
		}
	}

	res := make([]S, 0, len(a))
	for id, v := range m1 {
		if _, found := m2[id]; !found {
			res = append(res, v)
		}
	}

	return res
}
