package domainx

type ConType string

const (
	Mysql ConType = "mysql"
	Redis ConType = "redis"
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
