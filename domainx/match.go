package domainx

import "github.com/spf13/cast"

type MatchType string

const (
	MEq    MatchType = "="
	MEqr   MatchType = "="
	MLt    MatchType = "<"
	MLte   MatchType = "<="
	MGt    MatchType = ">"
	MGte   MatchType = ">="
	MNE    MatchType = "!="
	MLIKE  MatchType = "like"
	MIN    MatchType = "in"
	MNOTIN MatchType = "not in"
)

type Match struct {
	Field string
	Value interface{}
	Type  MatchType
}

type Matches []Match

func (m *Matches) Add(field string, value interface{}, t MatchType, ignore ...bool) *Matches {
	if value == nil {
		return m
	}
	if len(ignore) == 0 || !ignore[0] {
		// 根据类型判断是否取值 字符串判断非"" 数字判断非0 数组判断长度大于0 map判断长度大于0 struct判断是否有值
		switch value.(type) {
		case string:
			if value.(string) == "" {
				return m
			}
		case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
			if cast.ToInt64(value) == 0 {
				return m
			}
		case float32, float64:
			if value.(float64) == 0 {
				return m
			}
		case []interface{}:
			if len(value.([]interface{})) == 0 {
				return m
			}
		case map[string]interface{}:
			if len(value.(map[string]interface{})) == 0 {
				return m
			}
		case struct{}:
			if value == struct{}{} {
				return m
			}
		}
	}
	return m.AddMatch(&Match{Field: field, Value: value, Type: t})
}

func (m *Matches) Like(field string, value string, ignore ...bool) *Matches {
	return m.Add(field, value, MLIKE, ignore...)
}

func (m *Matches) Eq(field string, value interface{}, ignore ...bool) *Matches {
	return m.Add(field, value, MEq, ignore...)
}

func (m *Matches) Lt(field string, value interface{}, ignore ...bool) *Matches {
	return m.Add(field, value, MLt, ignore...)
}

func (m *Matches) Lte(field string, value interface{}, ignore ...bool) *Matches {
	return m.Add(field, value, MLte, ignore...)
}

func (m *Matches) Gt(field string, value interface{}, ignore ...bool) *Matches {
	return m.Add(field, value, MGt, ignore...)
}

func (m *Matches) Gte(field string, value interface{}, ignore ...bool) *Matches {
	return m.Add(field, value, MGte, ignore...)
}

func (m *Matches) Ne(field string, value interface{}, ignore ...bool) *Matches {
	return m.Add(field, value, MNE, ignore...)
}

func (m *Matches) In(field string, value interface{}, ignore ...bool) *Matches {
	return m.Add(field, value, MIN, ignore...)
}

func (m *Matches) NotIn(field string, value interface{}, ignore ...bool) *Matches {
	return m.Add(field, value, MNOTIN, ignore...)
}

func (m *Matches) AddMatch(match *Match) *Matches {
	if match.Value == nil {
		return m
	}
	*m = append(*m, *match)
	return m
}

func (m *Matches) AddMatches(matches *Matches) *Matches {
	if matches == nil {
		return m
	}
	*m = append(*m, *matches...)
	return m
}