package dmod

import (
	"errors"
	"fmt"

	"reflect"
	"strings"

	"github.com/jinzhu/copier"
)

var (
	TypeOfModleField = reflect.TypeOf((*ModelField)(nil)).Elem()
	TypeOfError      = reflect.TypeOf((*error)(nil)).Elem()
)

type ModelField struct {
	name       string
	fieldValue reflect.Value
}

func (p *ModelField) Name() string {
	return p.name
}

func (p *ModelField) Field(name string) *ModelField {

	if len(name) == 0 {
		return p
	}

	if !p.fieldValue.IsValid() {
		return nil
	}

	names := strings.Split(name, ".")

	m := &ModelField{
		name:       p.name + "." + names[0],
		fieldValue: indirect(p.fieldValue).FieldByName(name),
	}

	newNames := strings.Join(names[1:], ".")

	if len(newNames) == 0 {
		return m
	}

	return m.Field(newNames)
}

func (p *ModelField) Value(v interface{}) (err error) {

	if !p.fieldValue.IsValid() {
		return
	}

	err = copier.Copy(v, p.fieldValue.Interface())

	return
}

func (p *ModelField) Interface() interface{} {

	if !p.fieldValue.IsValid() {
		return nil
	}

	return p.fieldValue.Addr().Interface()
}

func (p *ModelField) Set(value interface{}) (err error) {
	if !p.fieldValue.IsValid() {
		return errors.New("field value not valid")
	}

	if !p.fieldValue.CanAddr() {
		return errors.New("using unaddressable value")
	}

	reflectValue, ok := value.(reflect.Value)
	if !ok {
		reflectValue = reflect.ValueOf(value)
	}

	fieldValue := p.fieldValue
	if reflectValue.IsValid() {
		if reflectValue.Type().ConvertibleTo(fieldValue.Type()) {
			fieldValue.Set(reflectValue.Convert(fieldValue.Type()))
		} else {
			if fieldValue.Kind() == reflect.Ptr {
				if fieldValue.IsNil() {
					fieldValue.Set(reflect.New(fieldValue.Type().Elem()))
				}
				fieldValue = fieldValue.Elem()
			}

			if reflectValue.Type().ConvertibleTo(fieldValue.Type()) {
				fieldValue.Set(reflectValue.Convert(fieldValue.Type()))
			} else {
				err = fmt.Errorf("could not convert argument of field %s from %s to %s", p.Name, reflectValue.Type(), fieldValue.Type())
			}
		}
	} else {
		p.fieldValue.Set(reflect.Zero(p.fieldValue.Type()))
	}

	return err
}

func (p *ModelField) Call(fn interface{}) (err error) {

	if fn == nil {
		err = errors.New("the model filed Call Args fn is nil")
		return
	}

	if !p.fieldValue.IsValid() {
		err = errors.New("this field is not valid")
		return
	}

	valFn := reflect.ValueOf(fn)
	if valFn.Kind() != reflect.Func {
		err = errors.New("the model filed Call Args fn should be func")
		return
	}

	if valFn.Type().NumIn() != 1 && valFn.Type().NumIn() != 2 {
		err = errors.New("fn's args should be `func(Any) error` OR `func(*ModleField) error` OR `func(Any, *ModleField) error`")
		return
	}

	if valFn.Type().NumOut() > 1 {
		err = errors.New("fn's return value should be void OR error")
		return
	}

	if valFn.Type().NumOut() == 1 {
		if valFn.Type().Out(0) != TypeOfError {
			err = errors.New("fn's return value should be void OR error")
			return
		}
	}

	var argTypes []reflect.Type
	var argTypesEle []bool

	for i := 0; i < valFn.Type().NumIn(); i++ {
		inType := valFn.Type().In(i)
		if inType.Kind() == reflect.Ptr {
			argTypes = append(argTypes, inType.Elem())
			argTypesEle = append(argTypesEle, true)
		} else {
			argTypes = append(argTypes, inType)
			argTypesEle = append(argTypesEle, false)
		}
	}

	if len(argTypes) == 2 {
		if argTypes[0] == argTypes[1] {
			err = errors.New("arg type is equal")
			return
		} else if argTypes[0] != TypeOfModleField && argTypes[1] != TypeOfModleField {
			err = errors.New("one arg should be type of *ModelField")
			return
		}
	}

	var argVals []reflect.Value
	for i := 0; i < len(argTypes); i++ {
		if argTypes[i] == TypeOfModleField {
			argVals = append(argVals, reflect.ValueOf(p))
		} else {

			argV := reflect.New(argTypes[i])

			err = copier.Copy(argV.Interface(), p.fieldValue.Interface())
			if err != nil {
				return
			}

			if !argTypesEle[i] {
				argV = argV.Elem()
			}

			argVals = append(argVals, argV)
		}
	}

	retVals := valFn.Call(argVals)

	if len(retVals) == 1 {
		if retVals[0].IsValid() && !retVals[0].IsNil() {
			err = retVals[0].Interface().(error)
			return
		}
	}

	return
}
