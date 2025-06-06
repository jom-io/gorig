package domainx

type Sort struct {
	Field  string
	Asc    bool
	Prefix string
}

type Sorts []*Sort

func (s *Sorts) AddSort(field string, asc bool, prefix ...string) *Sorts {
	if s == nil {
		s = &Sorts{}
	}
	px := ""
	if len(prefix) > 0 {
		for _, v := range prefix {
			px += v + "."
		}
	}
	if len(px) > 0 && px[len(px)-1] == '.' {
		px = px[:len(px)-1]
	}
	//*c.gSort = append(*c.gSort, &Sort{Field: field, Asc: asc, Prefix: pre})
	*s = append(*s, &Sort{Field: field, Asc: asc, Prefix: px})
	return s
}

func (s *Sorts) SetSort(sort ...*Sort) *Sorts {
	*s = sort
	return s
}

func (s *Sorts) GetSort() []*Sort {
	return *s
}
