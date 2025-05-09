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
	pre := ""
	if len(prefix) > 0 {
		for _, v := range prefix {
			pre += v + "."
		}
	}
	if len(pre) > 0 && pre[len(pre)-1] == '.' {
		pre = pre[:len(pre)-1]
	}
	//*c.gSort = append(*c.gSort, &Sort{Field: field, Asc: asc, Prefix: pre})
	*s = append(*s, &Sort{Field: field, Asc: asc, Prefix: pre})
	return s
}

func (s *Sorts) SetSort(sort ...*Sort) *Sorts {
	*s = sort
	return s
}

func (s *Sorts) GetSort() []*Sort {
	return *s
}
