package dmod

import (
	"encoding/json"
	"reflect"
	"strings"
	"sync"
)

type Model struct {
	name string

	fields       []Field
	structOf     reflect.Type
	structFields []reflect.StructField

	combineMap   map[string]interface{}
	combineVal   map[string]reflect.Value
	combineField map[string]reflect.StructField

	builder StructBuilder

	config ModelConfig

	locker sync.Mutex
}

func (p *Model) Name() string {
	return p.name
}

func (p *Model) Fields() []Field {
	return p.fields
}

func (p *Model) Dump() string {
	conf := p.config
	conf.reset()

	dumpData, _ := json.MarshalIndent(conf, "", "    ")
	return string(dumpData)
}

func (p *Model) Combine(combineMap map[string]interface{}) (err error) {
	p.locker.Lock()
	defer p.locker.Unlock()

	p.combineMap = combineMap

	return
}

func (p *Model) Delete(name string) (deleted bool, err error) {
	var fields []Field

	fields, deleted = deleteField(name, p.fields)

	if !deleted {
		return
	}

	err = p.updateStruct(fields)

	if err != nil {
		deleted = false
	}

	return
}

func (p *Model) Insert(onFiled string, fields []Field) (effect int, err error) {

	succesCount := 0
	var newFields = p.fields

	for i := 0; i < len(fields); i++ {
		var inserted bool
		newFields, inserted = insertField(onFiled, fields[i], newFields)
		if inserted {
			succesCount++
		}
	}

	if succesCount == 0 {
		return
	}

	err = p.updateStruct(newFields)
	if err == nil {
		effect = succesCount
	}

	return
}

func (p *Model) Update(onFiled string, field Field) (updated bool, err error) {

	var fields []Field
	fields, updated = updateField(onFiled, field, p.fields)
	if !updated {
		return
	}

	err = p.updateStruct(fields)

	if err != nil {
		updated = false
	}

	return
}

func (p *Model) updateStruct(fields []Field) (err error) {
	p.locker.Lock()
	defer p.locker.Unlock()

	sfileds, err := p.builder.Build(fields, p.combineMap)

	if err != nil {
		return
	}

	p.structFields = sfileds
	p.fields = fields

	p.structOf = reflect.StructOf(sfileds)

	return
}

func (p *Model) Type() reflect.Type {
	return p.structOf
}

func (p *Model) New(values ...interface{}) interface{} {
	st := reflect.New(p.structOf)

	p.copyModel(st, values...)

	for name, v := range p.combineMap {
		p.updateCombine(name, st, reflect.ValueOf(v))
	}

	p.updateSlice(st)

	return st.Interface()
}

func (p *Model) updateCombine(name string, st reflect.Value, newVal reflect.Value) {
	name = strings.TrimSpace(name)
	name = strings.TrimPrefix(name, ".")

	fieldNames := strings.SplitN(name, ".", 2)

	if len(fieldNames) == 1 && len(fieldNames[0]) == 0 {
		st.Elem().Field(0).Set(newVal)
		return
	}

	for i := 0; i < st.Elem().NumField(); i++ {
		field := st.Elem().FieldByName(fieldNames[0])

		if !field.IsValid() || field.Kind() != reflect.Struct {
			continue
		}

		if len(fieldNames) == 1 {
			field.Field(0).Set(newVal)
			return
		} else if len(fieldNames) > 1 {
			p.updateCombine(fieldNames[1], st, newVal)
		}
	}
}

func (p *Model) updateSlice(st reflect.Value) {

	iST := indirect(st)
	for i := 0; i < iST.NumField(); i++ {

		fieldI := iST.Field(i)

		if !fieldI.IsValid() {
			continue
		}

		if fieldI.Kind() == reflect.Slice {
			s := reflect.New(fieldI.Type())
			fieldI.Set(s.Elem())
		} else if fieldI.Kind() == reflect.Struct {
			p.updateSlice(fieldI)
		}
	}
}

func (p *Model) Field(v interface{}, name string) *ModelField {
	if v == nil {
		return nil
	}

	valV := reflect.ValueOf(v)

	name = strings.TrimPrefix(name, ".")

	if len(name) == 0 {
		return &ModelField{
			name:       ".",
			fieldValue: valV,
		}
	}

	names := strings.Split(name, ".")

	m := &ModelField{
		name:       "." + names[0],
		fieldValue: reflect.Indirect(valV).FieldByName(names[0]),
	}

	newNames := strings.Join(names[1:], ".")

	m = m.Field(newNames)

	return m
}

func (p *Model) copyModel(st reflect.Value, values ...interface{}) {

	if len(values) == 0 {
		return
	}

	iST := st.Elem()
	value0 := reflect.ValueOf(values[0])
	typ0 := value0.Type()

	for i := 0; i < typ0.NumField(); i++ {
		field := iST.FieldByName(typ0.Field(i).Name)
		if field.IsValid() {
			field.Set(value0.Field(i))
		}
	}

	for i := 1; i < len(values); i++ {
		val := reflect.ValueOf(values[i])
		typ := val.Type()
		field := iST.FieldByName(typ.Name())

		if !field.IsValid() {
			continue
		}

		filedType := field.Type()

		for j := 0; j < typ.NumField(); j++ {
			fieldDeep := field.FieldByName(filedType.Field(j).Name)
			if fieldDeep.IsValid() {
				fieldDeep.Set(val.Field(j))
			}
		}
	}

	return
}

func (p *Model) String() string {
	return p.name
}
