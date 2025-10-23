package domainx

type ConType string

const (
	Mysql ConType = "mysql"
	Mongo ConType = "mongo"
)

func (c ConType) String() string {
	return string(c)
}

func (c *Con) GetConType() ConType {
	if c.ConType == "" {
		return Mysql
	}
	return c.ConType
}

func (c *Con) GetConStr() string {
	switch c.GetConType() {
	case Mysql:
		return "Mysql"
	case Mongo:
		return "mongo"
	}
	return ""
}
