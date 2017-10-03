package dmod

import (
	"fmt"
)

type ModelConfig struct {
	Name    string   `json:"name"`
	Fields  []Field  `json:"fields,omitempty"`
	Extends []string `json:"extends,omitempty"`

	filepath       string
	extendsUpdated bool

	originalFields []Field
}

func (p *ModelConfig) reset() {
	p.Fields = p.originalFields
	p.extendsUpdated = false

	for i := 0; i < len(p.Fields); i++ {
		fieldRefReset(&p.Fields[i])
	}
}

type ModelsConfig struct {
	Models []ModelConfig `json:"models"`
}

func modelExtendsUpdate(models map[string]*ModelConfig, model *ModelConfig) {

	if len(model.Extends) == 0 || model.extendsUpdated {
		return
	}

	model.Fields = model.originalFields

	for i := 0; i < len(model.Extends); i++ {
		extendModel, exist := models[model.Extends[i]]
		if !exist {
			panic(fmt.Sprintf("extend model not exist, model: %s, path: %s, ref: %s ", model.Name, model.Extends[i], model.filepath))
		}

		model.Fields = append(model.Fields, extendModel.Fields...)
	}

	model.extendsUpdated = true
}

func fieldRefUpdate(models map[string]*ModelConfig, model *ModelConfig, field *Field) {

	if len(field.Ref) == 0 || field.refUpdated {
		return
	}

	refModel, exist := models[field.Ref]
	if !exist {
		panic(fmt.Sprintf("ref field not exist, model: %s, field: %s, path: %s, ref: %s ", model.Name, field.Name, model.filepath, field.Ref))
	}

	field.Children = refModel.Fields
	field.refUpdated = true
	field.filepath = model.filepath

	for i := 0; i < len(field.Children); i++ {
		fieldRefUpdate(models, model, &field.Children[i])
	}
}

func fieldRefReset(field *Field) {

	if !field.refUpdated {
		return
	}

	field.reset()

	for i := 0; i < len(field.Children); i++ {
		fieldRefReset(&field.Children[i])
	}
}
