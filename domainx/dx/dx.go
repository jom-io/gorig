package dx

import (
	"context"
	"github.com/jom-io/gorig/apix/load"
	"github.com/jom-io/gorig/domainx"
	"github.com/jom-io/gorig/utils/errors"
)

type (
	dx[T any] struct {
		ctx     context.Context
		complex *domainx.Complex[T]
		matches *domainx.Matches
	}

	DTable interface {
		DConfig() (conType domainx.ConType, dbName string, tableName string)
	}

	DQuery[T any] interface {
		WithContext(ctx context.Context) DQuery[T]

		Complex() *domainx.Complex[T]
		GetData() *T
		GetCon() *domainx.Con
		SetID(id int64)
		GetID() domainx.ID
		GenerateID() DQuery[T]
		isNil() bool
		IsZero() bool

		WithID(id int64) DQuery[T]
		// Eq Ne Gt Gte Lt Lte Like In NotIn ignore is used to ignore the field empty check
		Eq(field string, value interface{}, ignore ...bool) DQuery[T]
		Ne(field string, value interface{}, ignore ...bool) DQuery[T]
		Gt(field string, value interface{}, ignore ...bool) DQuery[T]
		Gte(field string, value interface{}, ignore ...bool) DQuery[T]
		Lt(field string, value interface{}, ignore ...bool) DQuery[T]
		Lte(field string, value interface{}, ignore ...bool) DQuery[T]
		Like(field string, value string, ignore ...bool) DQuery[T]
		In(field string, value interface{}, ignore ...bool) DQuery[T]
		NotIn(field string, value interface{}, ignore ...bool) DQuery[T]
		NEmpty(field string) DQuery[T]
		Near(latField, lngField string, lat, lng, distance float64) DQuery[T]
		NearLoc(localField string, lat, lng, distance float64) DQuery[T]
		AddMatch(m *domainx.Match) DQuery[T]
		AddMatches(ms *domainx.Matches) DQuery[T]
		Sort(field string, asc ...bool) DQuery[T]

		Save(t ...*T) (id int64, err *errors.Error)
		checkMatches() *errors.Error
		Update(field string, value any) *errors.Error
		Updates(data map[string]interface{}) *errors.Error
		Delete() *errors.Error
		Get() (*domainx.Complex[T], *errors.Error)
		Find() (domainx.ComplexList[T], *errors.Error)
		Count() (int64, *errors.Error)
		Sum(field string) (float64, *errors.Error)
		Page(page *load.Page) (*load.PageRespT[*domainx.Complex[T]], *errors.Error)
	}
)

func On[T any, PT interface {
	*T
	DTable
}](ctx context.Context, t ...*T) DQuery[T] {
	var inst T
	if len(t) > 0 && any(t[0]) != nil && t[0] != nil {
		inst = *t[0]
	} else {
		inst = *new(T)
	}

	ptr := PT(&inst)

	conType, dbName, TableName := ptr.DConfig()
	return &dx[T]{
		ctx:     ctx,
		complex: domainx.CreateComplex(ctx, conType, dbName, TableName, &inst),
		matches: domainx.NewMatches(),
	}
}

func (d *dx[T]) WithContext(ctx context.Context) DQuery[T] {
	d.ctx = ctx
	if d.complex != nil {
		d.complex.Con.WithContext(ctx)
	}
	return d
}

func (d *dx[T]) Complex() *domainx.Complex[T] {
	return d.complex
}

func (d *dx[T]) GetData() *T {
	return d.complex.Data
}

func (d *dx[T]) GetCon() *domainx.Con {
	return d.complex.Con
}

func (d *dx[T]) SetID(id int64) {
	d.complex.Con.SetID(id)
}

func (d *dx[T]) GetID() domainx.ID {
	return d.complex.Con.GetID()
}

func (d *dx[T]) WithID(id int64) DQuery[T] {
	d.SetID(id)
	return d
}

func (d *dx[T]) GenerateID() DQuery[T] {
	d.complex.Con.GenerateID()
	return d
}

func (d *dx[T]) isNil() bool {
	return d.complex == nil || d.complex.Con == nil || d.GetData() == nil
}

func (d *dx[T]) IsZero() bool {
	return !d.isNil() && d.complex.GetID().IsZero()
}

func (d *dx[T]) Eq(field string, value interface{}, ignore ...bool) DQuery[T] {
	d.matches.Eq(field, value, ignore...)
	return d
}

func (d *dx[T]) Ne(field string, value interface{}, ignore ...bool) DQuery[T] {
	d.matches.Ne(field, value, ignore...)
	return d
}

func (d *dx[T]) Gt(field string, value interface{}, ignore ...bool) DQuery[T] {
	d.matches.Gt(field, value, ignore...)
	return d
}

func (d *dx[T]) Gte(field string, value interface{}, ignore ...bool) DQuery[T] {
	d.matches.Gte(field, value, ignore...)
	return d
}

func (d *dx[T]) Lt(field string, value interface{}, ignore ...bool) DQuery[T] {
	d.matches.Lt(field, value, ignore...)
	return d
}

func (d *dx[T]) Lte(field string, value interface{}, ignore ...bool) DQuery[T] {
	d.matches.Lte(field, value, ignore...)
	return d
}

func (d *dx[T]) Like(field string, value string, ignore ...bool) DQuery[T] {
	d.matches.Like(field, value, ignore...)
	return d
}

func (d *dx[T]) In(field string, value interface{}, ignore ...bool) DQuery[T] {
	d.matches.In(field, value, ignore...)
	return d
}

func (d *dx[T]) NotIn(field string, value interface{}, ignore ...bool) DQuery[T] {
	d.matches.NotIn(field, value, ignore...)
	return d
}

func (d *dx[T]) NEmpty(field string) DQuery[T] {
	d.matches.NEmpty(field)
	return d
}

func (d *dx[T]) Near(latField, lngField string, lat, lng, distance float64) DQuery[T] {
	d.matches.Near(latField, lngField, lat, lng, distance)
	return d
}

func (d *dx[T]) NearLoc(localField string, lat, lng, distance float64) DQuery[T] {
	d.matches.NearLoc(localField, lat, lng, distance)
	return d
}

func (d *dx[T]) AddMatch(m *domainx.Match) DQuery[T] {
	d.matches.AddMatch(m)
	return d
}

func (d *dx[T]) AddMatches(ms *domainx.Matches) DQuery[T] {
	d.matches.AddMatches(ms)
	return d
}

func (d *dx[T]) Sort(field string, asc ...bool) DQuery[T] {
	if field == "" {
		return d
	}
	d.complex.Sort.AddSort(field, len(asc) > 0 && asc[0])
	return d
}

func (d *dx[T]) Save(t ...*T) (id int64, err *errors.Error) {
	if len(t) > 0 && any(t[0]) != nil {
		d.complex.Data = t[0]
	}
	return domainx.Save(d.complex.Con, d.complex, 0)
}

func (d *dx[T]) checkMatches() *errors.Error {
	if d.IsZero() && (d.matches == nil || len(*d.matches) == 0) {
		return errors.Sys("id is zero or matches not set")
	}
	if d.matches != nil && len(*d.matches) == 0 {
		return errors.Sys("matches cannot be empty")
	}
	return nil
}

func (d *dx[T]) Update(field string, value any) *errors.Error {
	if field == "" {
		return errors.Sys("field name cannot be empty")
	}
	if value == nil {
		return errors.Sys("value cannot be nil")
	}
	if !d.IsZero() {
		return domainx.UpdatePart(d.complex.Con, d.GetID().Int64(), map[string]interface{}{field: value})
	}

	if err := d.checkMatches(); err != nil {
		return err
	}
	return domainx.UpdateByMatch(d.complex.Con, *d.matches, map[string]interface{}{field: value})
}

func (d *dx[T]) Updates(data map[string]interface{}) *errors.Error {
	if len(data) == 0 {
		return errors.Sys("data map cannot be empty")
	}
	if !d.IsZero() {
		return domainx.UpdatePart(d.complex.Con, d.GetID().Int64(), data)
	}

	if err := d.checkMatches(); err != nil {
		return err
	}
	return domainx.UpdateByMatch(d.complex.Con, *d.matches, data)
}

func (d *dx[T]) Delete() *errors.Error {
	if !d.IsZero() {
		return domainx.Delete(d.complex.Con, d)
	}

	if err := d.checkMatches(); err != nil {
		return err
	}
	return domainx.DeleteByMatch(d.complex.Con, *d.matches)
}

func (d *dx[T]) Get() (*domainx.Complex[T], *errors.Error) {
	if !d.IsZero() {
		if err := domainx.GetByID(d.complex.Con, d.GetID().Int64(), d.complex); err != nil {
			return nil, err
		}
		return d.complex, nil
	}

	if err := d.checkMatches(); err != nil {
		return nil, err
	}
	if err := domainx.GetByMatch(d.complex.Con, *d.matches, d.complex); err != nil {
		return nil, err
	}
	return d.complex, nil
}

func (d *dx[T]) Find() (domainx.ComplexList[T], *errors.Error) {
	if err := d.checkMatches(); err != nil {
		return nil, err
	}
	var result []*domainx.Complex[T]
	if err := domainx.FindByMatch(d.complex.Con, *d.matches, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (d *dx[T]) Count() (int64, *errors.Error) {
	count, err := domainx.CountByMatch(d.complex.Con, *d.matches)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (d *dx[T]) Sum(field string) (float64, *errors.Error) {
	if field == "" {
		return 0, errors.Sys("field name cannot be empty")
	}
	sum, err := domainx.SumByMatch(d.complex.Con, *d.matches, field)
	if err != nil {
		return 0, err
	}
	return sum, nil
}

func (d *dx[T]) Page(page *load.Page) (*load.PageRespT[*domainx.Complex[T]], *errors.Error) {
	if page == nil {
		return nil, errors.Sys("page cannot be nil")
	}
	resp := &load.PageRespT[*domainx.Complex[T]]{Result: &[]*domainx.Complex[T]{}}
	if err := domainx.FindByPageMatchT(d.complex.Con, *d.matches, page, resp, resp.Result); err != nil {
		return nil, err
	}
	return resp, nil
}
