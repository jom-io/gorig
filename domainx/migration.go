package domainx

type IdxType string

const (
	Unique    IdxType = "unique"
	Idx       IdxType = "idx"
	Spatial2D IdxType = "2d"
)

type Index struct {
	IdxType IdxType
	Fields  []string
	IdxName string
}

var MigrationList []*Migration

type Migration struct {
	DBFunc func() ConTable
	Index  []Index
}

type IndexFunc func() []Index
