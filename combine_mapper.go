package dmod

import (
	"fmt"
	"sync"
)

type CombineMapper interface {
	Register(modelID string, fn MapperFunc)
	Unregister(modelID string)
	GetMapper(modelID string) (mapper MapperFunc, exist bool)
}

type MapperFunc func(modelID string, fields []Field) map[string]interface{}

type BasicMapper struct {
	sync.Mutex
	maps map[string]MapperFunc
}

func NewBasicMapper() *BasicMapper {
	return &BasicMapper{
		maps: make(map[string]MapperFunc),
	}
}

func (p *BasicMapper) Register(id string, fn MapperFunc) {
	if len(id) == 0 {
		return
	}

	if fn == nil {
		return
	}

	p.Lock()
	defer p.Unlock()

	_, exist := p.maps[id]

	if exist {
		panic(fmt.Sprintf("modle id of %s already registered", id))
	}

	p.maps[id] = fn
}

func (p *BasicMapper) Unregister(id string) {
	p.Lock()
	defer p.Unlock()

	delete(p.maps, id)
}

func (p *BasicMapper) GetMapper(id string) (mapper MapperFunc, exist bool) {
	mapper, exist = p.maps[id]
	return mapper, exist
}
