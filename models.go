package dmod

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"

	"github.com/sirupsen/logrus"
)

type Models struct {
	locker sync.Mutex

	modelsInstance map[string]*Model
	combineMapper  CombineMapper
	builder        StructBuilder

	modelsConfig map[string]*ModelConfig
}

type ModelsOption func(*Models) error

func NewModels(opts ...ModelsOption) (models *Models, err error) {
	m := &Models{
		modelsInstance: make(map[string]*Model),
		combineMapper:  NewBasicMapper(),
		builder:        defaultBuilder,
	}

	for i := 0; i < len(opts); i++ {
		err = opts[i](m)
		if err != nil {
			return
		}
	}

	models = m

	return
}

func ModelsOptBaseMapper(mapper CombineMapper) ModelsOption {
	return func(m *Models) error {
		m.combineMapper = mapper
		return nil
	}
}

func ModelsOptBuilder(builder StructBuilder) ModelsOption {
	return func(m *Models) error {
		m.builder = builder
		return nil
	}
}

func (p *Models) Flush() {
	p.locker.Lock()
	p.modelsConfig = map[string]*ModelConfig{}
	p.modelsInstance = map[string]*Model{}
	p.locker.Unlock()
}

func (p *Models) Dump() string {

	allModels := map[string]ModelConfig{}

	for k, v := range p.modelsInstance {
		copyConf := v.config
		copyConf.reset()
		allModels[k] = copyConf
	}

	dumpData, _ := json.MarshalIndent(allModels, "", "    ")
	return string(dumpData)
}

func (p *Models) DeleteModel(name string) bool {

	p.locker.Lock()
	defer p.locker.Unlock()

	_, exist := p.modelsConfig[name]
	if !exist {
		return false
	}

	delete(p.modelsConfig, name)
	delete(p.modelsInstance, name)

	return true
}

func (p *Models) LoadModels(modleSchemas []string) (err error) {

	allModels := map[string]*ModelConfig{}

	for k, v := range p.modelsConfig {
		copyModel := *v
		copyModel.reset()
		allModels[k] = &copyModel
	}

	for _, schema := range modleSchemas {

		modelConfig := ModelConfig{}

		err = json.Unmarshal([]byte(schema), &modelConfig)
		if err != nil {
			return
		}

		modelConfig.originalFields = modelConfig.Fields

		_, existModel := allModels[modelConfig.Name]
		if existModel {
			logrus.WithField("model", modelConfig.Name).Warnln("model already exist")
		}

		allModels[modelConfig.Name] = &modelConfig
		logrus.WithField("model", modelConfig.Name).Debug("model loaded")
	}

	if err != nil {
		return
	}

	for _, model := range allModels {
		for i := 0; i < len(model.Fields); i++ {
			fieldRefUpdate(allModels, model, &model.Fields[i])
		}
	}

	for _, model := range allModels {
		modelExtendsUpdate(allModels, model)
	}

	p.modelsConfig = allModels

	for _, model := range allModels {
		_, err = p.SetModel(*model)
		if err != nil {
			return
		}
	}

	return
}

func (p *Models) LoadFromFiles(files ...string) (err error) {

	allModels := map[string]*ModelConfig{}

	for k, v := range p.modelsConfig {
		copyModel := *v
		copyModel.reset()
		allModels[k] = &copyModel
	}

	for _, file := range files {

		logrus.WithField("file", file).Debug("begin load")

		var data []byte

		data, err = ioutil.ReadFile(file)
		if err != nil {
			return
		}

		modelConfig := ModelConfig{}

		err = json.Unmarshal(data, &modelConfig)
		if err != nil {
			return
		}

		modelConfig.filepath = file
		modelConfig.originalFields = modelConfig.Fields

		_, existModel := allModels[modelConfig.Name]
		if existModel {
			logrus.WithField("model", modelConfig.Name).WithField("file", file).Warnln("model already exist")
		}

		allModels[modelConfig.Name] = &modelConfig
		logrus.WithField("file", file).WithField("model", modelConfig.Name).Debug("model loaded")
	}

	if err != nil {
		return
	}

	for _, model := range allModels {
		for i := 0; i < len(model.Fields); i++ {
			fieldRefUpdate(allModels, model, &model.Fields[i])
		}
	}

	for _, model := range allModels {
		modelExtendsUpdate(allModels, model)
	}

	p.modelsConfig = allModels

	for _, model := range allModels {
		_, err = p.SetModel(*model)
		if err != nil {
			return
		}
	}

	return
}

func (p *Models) LoadFromDir(dir string) (err error) {

	allModels := map[string]*ModelConfig{}

	walkFn := func(path string, info os.FileInfo, e error) (walkErr error) {

		if info.IsDir() {
			if strings.HasPrefix(filepath.Base(path), ".") {
				return filepath.SkipDir
			}
			return
		}

		if filepath.Ext(path) != ".json" {
			return
		}

		relPath, _ := filepath.Rel(dir, path)

		logrus.WithField("file", relPath).Debug("begin load")

		data, walkErr := ioutil.ReadFile(path)
		if walkErr != nil {
			return
		}

		modelConfig := ModelConfig{}

		walkErr = json.Unmarshal(data, &modelConfig)
		if walkErr != nil {
			return
		}

		modelConfig.filepath = path
		modelConfig.originalFields = modelConfig.Fields

		_, existModel := allModels[modelConfig.Name]
		if existModel {
			logrus.WithField("model", modelConfig.Name).WithField("file", path).Warnln("model already exist")
		}

		allModels[modelConfig.Name] = &modelConfig
		logrus.WithField("file", relPath).WithField("model", modelConfig.Name).Debug("model loaded")

		return
	}

	err = filepath.Walk(dir, walkFn)

	if err != nil {
		return
	}

	for _, model := range allModels {
		for i := 0; i < len(model.Fields); i++ {
			fieldRefUpdate(allModels, model, &model.Fields[i])
		}
	}

	for _, model := range allModels {
		modelExtendsUpdate(allModels, model)
	}

	p.modelsConfig = allModels

	for _, model := range allModels {
		_, err = p.SetModel(*model)
		if err != nil {
			return
		}
	}

	return
}

func (p *Models) NewModel(config ModelConfig) (model *Model, err error) {

	_, exist := p.modelsInstance[config.Name]

	if exist {
		err = fmt.Errorf("model %s already exist", config.Name)
		return
	}

	m, err := p.SetModel(config)
	if err != nil {
		return
	}

	model = m

	for _, model := range p.modelsConfig {
		for i := 0; i < len(model.Fields); i++ {
			fieldRefUpdate(p.modelsConfig, model, &model.Fields[i])
		}
	}

	for _, model := range p.modelsConfig {
		modelExtendsUpdate(p.modelsConfig, model)
	}

	return
}

func (p *Models) GetModel(name string) (*Model, bool) {
	m, e := p.modelsInstance[name]
	return m, e
}

func (p *Models) Models() []*Model {
	var models []*Model

	for _, v := range p.modelsInstance {
		models = append(models, v)
	}

	return models
}

func (p *Models) SetModel(config ModelConfig) (model *Model, err error) {
	if len(config.Name) == 0 {
		err = fmt.Errorf("name is empty")
		return
	}

	p.locker.Lock()
	defer p.locker.Unlock()

	mapperFn, exist := p.CombineMapper().GetMapper(config.Name)
	var combineMap map[string]interface{}
	if exist {
		combineMap = mapperFn(config.Name, config.Fields)
	}

	var structFields []reflect.StructField
	structFields, err = p.builder.Build(config.Fields, combineMap)

	if err != nil {
		return
	}

	model = &Model{
		name:         config.Name,
		fields:       config.Fields,
		builder:      p.builder,
		structFields: structFields,
		combineMap:   combineMap,
		config:       config,
		structOf:     reflect.StructOf(structFields),
	}

	p.modelsInstance[config.Name] = model

	return
}

func (p *Models) Produce(model *Model, values ...interface{}) interface{} {
	if model == nil {
		return nil
	}

	return p.ProduceByName(model.name, values...)
}

func (p *Models) ProduceByName(name string, values ...interface{}) interface{} {
	if len(name) == 0 {
		return nil
	}

	model, exist := p.GetModel(name)
	if !exist {
		return nil
	}

	return model.New(values...)
}

func (p *Models) CombineMapper() CombineMapper {
	return p.combineMapper
}

func (p *Models) StructBuilder() StructBuilder {
	return p.builder
}
