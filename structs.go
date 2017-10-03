package dmod

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"
)

var (
	defaultBuilder = NewBuilder()
)

var (
	typeMap = map[string]reflect.Type{
		"string":      reflect.TypeOf((*string)(nil)).Elem(),
		"int":         reflect.TypeOf((*int)(nil)).Elem(),
		"int8":        reflect.TypeOf((*int8)(nil)).Elem(),
		"int32":       reflect.TypeOf((*int32)(nil)).Elem(),
		"int64":       reflect.TypeOf((*int64)(nil)).Elem(),
		"uint":        reflect.TypeOf((*uint)(nil)).Elem(),
		"uint8":       reflect.TypeOf((*uint8)(nil)).Elem(),
		"uint32":      reflect.TypeOf((*uint32)(nil)).Elem(),
		"uint64":      reflect.TypeOf((*uint64)(nil)).Elem(),
		"float32":     reflect.TypeOf((*float32)(nil)).Elem(),
		"float64":     reflect.TypeOf((*float64)(nil)).Elem(),
		"bool":        reflect.TypeOf((*bool)(nil)).Elem(),
		"byte":        reflect.TypeOf((*byte)(nil)).Elem(),
		"error":       reflect.TypeOf((*error)(nil)).Elem(),
		"struct":      reflect.TypeOf((*struct{})(nil)).Elem(),
		"time.Time":   reflect.TypeOf((*time.Time)(nil)).Elem(),
		"interface{}": reflect.TypeOf((*interface{})(nil)).Elem(),

		"*string":      reflect.TypeOf((*string)(nil)),
		"*int":         reflect.TypeOf((*int)(nil)),
		"*int8":        reflect.TypeOf((*int8)(nil)),
		"*int32":       reflect.TypeOf((*int32)(nil)),
		"*int64":       reflect.TypeOf((*int64)(nil)),
		"*uint":        reflect.TypeOf((*uint)(nil)),
		"*uint8":       reflect.TypeOf((*uint8)(nil)),
		"*uint32":      reflect.TypeOf((*uint32)(nil)),
		"*uint64":      reflect.TypeOf((*uint64)(nil)),
		"*float32":     reflect.TypeOf((*float32)(nil)),
		"*float64":     reflect.TypeOf((*float64)(nil)),
		"*bool":        reflect.TypeOf((*bool)(nil)),
		"*byte":        reflect.TypeOf((*byte)(nil)),
		"*error":       reflect.TypeOf((*error)(nil)),
		"*struct":      reflect.TypeOf((*struct{})(nil)),
		"*time.Time":   reflect.TypeOf((*time.Time)(nil)),
		"*interface{}": reflect.TypeOf((*interface{})(nil)),

		"map[string]string":      reflect.TypeOf((*[]map[string]string)(nil)).Elem(),
		"map[string]interface{}": reflect.TypeOf((*[]map[string]interface{})(nil)).Elem(),
	}
)

type StructBuilder interface {
	Build(fields []Field, combineMap map[string]interface{}) (structFields []reflect.StructField, err error)
	RegisterTypes(nameTypes ...NameType)
}

type Field struct {
	Children  []Field `json:"children,omitempty"`
	Name      string  `json:"name"`
	Type      string  `json:"type,omitempty"`
	Array     bool    `json:"array,omitempty"`
	Tag       string  `json:"tag,omitempty"`
	Anonymous bool    `json:"anonymous,omitempty"`
	Ref       string  `json:"ref,omitempty"`

	refUpdated       bool
	originalChildren []Field
	filepath         string
}

func (p *Field) reset() {
	p.refUpdated = false
	p.Children = p.originalChildren
}

type NameType struct {
	Name string
	Type interface{}
}

type Builder struct {
	registeredTypes map[string]reflect.Type

	locker sync.Mutex
}

func NewBuilder() StructBuilder {
	return &Builder{
		registeredTypes: make(map[string]reflect.Type),
	}
}

func (p *Builder) RegisterTypes(nameTypes ...NameType) {
	p.locker.Lock()
	defer p.locker.Unlock()

	for i := 0; i < len(nameTypes); i++ {
		t := reflect.TypeOf(nameTypes[i].Type)
		if t.Kind() == reflect.Ptr {
			p.registeredTypes[nameTypes[i].Name] = reflect.TypeOf(nameTypes[i].Type).Elem()
		} else {
			p.registeredTypes[nameTypes[i].Name] = reflect.TypeOf(nameTypes[i].Type)
		}
	}
}

func (p *Builder) Build(fields []Field, combineMap map[string]interface{}) (structFields []reflect.StructField, err error) {

	var sFields []reflect.StructField

	for i := 0; i < len(fields); i++ {
		var typ reflect.Type
		typ, err = p.buildStructFields("."+fields[i].Name, fields[i], combineMap)
		if err != nil {
			return
		}

		if fields[i].Array {

			// typ = reflect.SliceOf(reflect.PtrTo(typ))
			// fmt.Println(typ)

			typ = reflect.SliceOf(typ)
		}

		sFields = append(sFields,
			reflect.StructField{
				Name:      fields[i].Name,
				Type:      typ,
				Tag:       reflect.StructTag(fields[i].Tag),
				Anonymous: fields[i].Anonymous,
			})
	}

	base, existCombine := combineMap["."]

	if existCombine {
		baseTyp := reflect.TypeOf(base)
		sFields = append([]reflect.StructField{
			reflect.StructField{
				Name:      baseTyp.Name(),
				Type:      baseTyp,
				Anonymous: true,
			},
		}, sFields...)
	}

	structFields = sFields
	return
}

func (p *Builder) buildStructFields(name string, field Field, combineMap map[string]interface{}) (retType reflect.Type, err error) {

	if len(field.Children) == 0 {
		typ, exist := p.registeredTypes[field.Type]

		if !exist {
			typ, exist = typeMap[field.Type]
		}

		if !exist {
			err = fmt.Errorf("type %s not register", field.Type)
			return
		}

		retType = typ
		return
	}

	var childFields []reflect.StructField
	for i := 0; i < len(field.Children); i++ {
		var typ reflect.Type
		typ, err = p.buildStructFields(name+"."+field.Name, field.Children[i], combineMap)
		if err != nil {
			return
		}

		childFields = append(childFields,
			reflect.StructField{
				Name:      field.Children[i].Name,
				Type:      typ,
				Tag:       reflect.StructTag(field.Children[i].Tag),
				Anonymous: field.Children[i].Anonymous,
			})
	}

	base, existCombine := combineMap[name]

	if existCombine {
		baseTyp := reflect.TypeOf(base)
		childFields = append([]reflect.StructField{
			reflect.StructField{Name: baseTyp.Name(), Type: baseTyp, Anonymous: true},
		}, childFields...)
	}

	retType = reflect.StructOf(childFields)

	return
}

func insertField(name string, newField Field, fields []Field) ([]Field, bool) {
	name = strings.TrimSpace(name)
	name = strings.TrimPrefix(name, ".")

	fieldNames := strings.SplitN(name, ".", 2)

	if len(fieldNames) == 1 && len(fieldNames[0]) == 0 {
		return append(fields, newField), true
	}

	inserted := false

	for i := 0; i < len(fields); i++ {
		if fields[i].Name == fieldNames[0] {

			if !fields[i].Array && (len(fields[i].Type) == 0 || fields[i].Type == "struct") {

				if len(fieldNames) == 1 {
					fields[i].Children = append(fields[i].Children, newField)
					return fields, true

				} else if len(fieldNames) > 1 {
					fields[i].Children, inserted = insertField(fieldNames[1], newField, fields[i].Children)

					if inserted {
						break
					}
				}
			}
		}
	}

	return fields, inserted
}

func updateField(name string, newField Field, fields []Field) ([]Field, bool) {
	name = strings.TrimSpace(name)
	name = strings.TrimPrefix(name, ".")

	fieldNames := strings.SplitN(name, ".", 2)

	if len(fieldNames) == 1 && len(fieldNames[0]) == 0 {
		return append(fields, newField), true
	}

	updated := false

	for i := 0; i < len(fields); i++ {
		if fields[i].Name == fieldNames[0] {

			if len(fieldNames) == 1 {
				fields[i] = newField
				return fields, true
			} else if len(fieldNames) > 1 {

				fields[i].Children, updated = updateField(fieldNames[1], newField, fields[i].Children)

				if len(fields[i].Children) == 0 {
					fields = append(fields[0:i], fields[i+1:]...)
				}

				if updated {
					break
				}
			}
		}
	}

	return fields, updated
}

func deleteField(name string, fields []Field) ([]Field, bool) {
	name = strings.TrimSpace(name)
	name = strings.TrimPrefix(name, ".")

	fieldNames := strings.SplitN(name, ".", 2)

	deleted := false

	for i := 0; i < len(fields); i++ {
		if fields[i].Name == fieldNames[0] {

			if len(fieldNames) == 1 {
				return append(fields[0:i], fields[i+1:]...), true
			} else if len(fieldNames) > 1 {

				fields[i].Children, deleted = deleteField(fieldNames[1], fields[i].Children)

				if len(fields[i].Children) == 0 {
					fields = append(fields[0:i], fields[i+1:]...)
				}

				if deleted {
					break
				}
			}
		}
	}

	return fields, deleted
}

func indirect(reflectValue reflect.Value) reflect.Value {
	for reflectValue.Kind() == reflect.Ptr {
		reflectValue = reflectValue.Elem()
	}
	return reflectValue
}

func indirectType(reflectType reflect.Type) reflect.Type {
	for reflectType.Kind() == reflect.Ptr || reflectType.Kind() == reflect.Slice {
		reflectType = reflectType.Elem()
	}
	return reflectType
}
