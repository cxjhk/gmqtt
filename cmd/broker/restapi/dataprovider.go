package restapi

import (
	"reflect"
	"fmt"
	"errors"
)

type DataProvider interface {
	Models() interface{}
	Count() int
	TotalCount() int
	Pagination() *Pagination
}



type SliceDataProvider struct {
	allModels interface{}
	totalCount int
	count int
	Pager *Pagination
}


type Pagination struct {
	Page int
	PageSize int
	TotalCount int
}


func (p *Pagination) PageCount() int{
	if p.PageSize < 1 {
		if p.TotalCount > 0 {
			return 1
		} else {
			return 0
		}
	}
	return (p.TotalCount + p.PageSize - 1) / p.PageSize

}


func (p *Pagination) Offset() int {
	if p.PageSize < 1 {
		return 0
	} else {
		return p.Page * p.PageSize
	}

}




func (cp *SliceDataProvider) SetModels(models interface{}) {
	if reflect.TypeOf(models).Kind() != reflect.Slice {
		panic(fmt.Sprintf("invalid models type,want slice, but got %s",reflect.TypeOf(models).Kind()))
	}
	cp.allModels = models
	cp.totalCount = reflect.ValueOf(models).Len()

}




func (cp *SliceDataProvider) Models(out interface{}) error {
	totalCount := cp.TotalCount()
	rv := reflect.ValueOf(out)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return errors.New(fmt.Sprintf("invalid value,want reflect.Ptr or not nil, got %s",reflect.TypeOf(out).Kind()))
	}
	e := rv.Elem()
	if totalCount == 0 {
		return nil
	}
	p := cp.Pagination()
	all := reflect.ValueOf(cp.allModels)
	if cp.Pagination() != nil {
		p.TotalCount = totalCount
		if p.Page + 1 > p.PageCount() {
			p.Page = p.PageCount() - 1
		}
		var limit int
		offset := p.Offset()
		if p.PageSize != 0 {
			if offset + p.PageSize  > totalCount {
				limit = totalCount
			} else {
				limit = offset + p.PageSize
			}
		} else {
			limit = p.TotalCount
		}
		cp.count = limit - offset
		s := reflect.MakeSlice(e.Type(),cp.count,cp.count)
		reflect.Copy(s, all.Slice(p.Offset(),limit))
		e.Set(all.Slice(p.Offset(),limit))
		return nil
	}
	e.Set(all)
	cp.count = totalCount
	return nil
}
func (cp *SliceDataProvider) Count() int {
	return cp.count
}
func (cp *SliceDataProvider) TotalCount() int {
	return cp.totalCount
}

func (cp *SliceDataProvider) Pagination() *Pagination{
	return cp.Pager
}



