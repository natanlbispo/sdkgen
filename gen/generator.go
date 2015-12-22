package gen

import (
	"fmt"
	"net/url"
	"os"
	"path"
	"reflect"
	"strings"
	"text/template"

	"github.com/alvaroloes/sdkgen/parser"
	"github.com/juju/errors"
)

//go:generate go-bindata -o templates_bindata.go -debug=$DEBUG -pkg $GOPACKAGE ../templates/...
//go:generate stringer -type=Language

var (
	ErrLangNotSupported = errors.New("language not supported")
)

type Language int

const (
	Android Language = iota
	ObjC
	Swift
)

const (
	templateDir       = "./templates"
	templateExt       = ".tpl"
	modelTemplatePath = "model"
	dirPermissions    = 0777
)

// Config contains the needed configuration for the generator
type Config struct {
	OutputDir     string
	ModelsRelPath string
	ApiName       string
	ApiPrefix     string
}

type templateData struct {
	Config           Config
	Api              *parser.Api
	CurrentModelInfo *modelInfo
	AllModelsInfo    map[string]*modelInfo
}

// Generator contains all the information needed to generate the SDK in a specific language
type Generator struct {
	gen        specificGenerator // The language specific generator
	api        *parser.Api
	modelsInfo map[string]*modelInfo // Contains processed information to generate the models
	config     Config
	tplDir     string
}

// The language specific generator interface
type specificGenerator interface {
	getFuncMap() template.FuncMap
}

func (g *Generator) Generate() error {
	// Extract the models info
	g.extractModelsInfo()

	generalTpls, err := template.ParseGlob(path.Join(g.tplDir, "*"+templateExt))
	if err != nil {
		return errors.Annotate(err, "when reading templates at "+g.tplDir)
	}

	modelTplDir := path.Join(g.tplDir, modelTemplatePath)
	modelTpls, err := template.ParseGlob(path.Join(modelTplDir, "*"+templateExt))
	if err != nil {
		return errors.Annotate(err, "when reading model templates at "+modelTplDir)
	}

	apiDir := path.Join(g.config.OutputDir, g.config.ApiName)
	modelsDir := path.Join(apiDir, g.config.ModelsRelPath)

	for _, tpl := range modelTpls.Templates() {
		for _, modelInfo := range g.modelsInfo {
			// Create the model file name
			modelFileName := modelInfo.Name
			tplFileName := tpl.Name()
			from := strings.Index(tplFileName, ".")
			to := strings.LastIndex(tplFileName, ".")
			if from > 0 && to > 0 {
				modelFileName += tplFileName[from:to]
			}

			// Create the full path and the file
			if err := os.MkdirAll(modelsDir, dirPermissions); err != nil {
				return errors.Annotatef(err, "when creating model %q", modelInfo.Name)
			}

			file, err := os.Create(path.Join(modelsDir, modelFileName))
			if err != nil {
				return errors.Annotatef(err, "when creating model %q", modelInfo.Name)
			}

			// Write the template to the file
			err = tpl.Funcs(funcMap).Execute(file, templateData{
				Config:           g.config,
				Api:              g.api,
				CurrentModelInfo: modelInfo,
				AllModelsInfo:    g.modelsInfo,
			})

			if err != nil {
				return errors.Annotatef(err, "when creating model %q", modelInfo.Name)
			}
		}
	}
	for _, tpl := range generalTpls.Templates() {
		fmt.Println(tpl.Name())
	}

	return nil
}

func (g *Generator) extractModelsInfo() {
	g.modelsInfo = map[string]*modelInfo{}
	for _, endpoint := range g.api.Endpoints {
		// Extract the resource whose information is contained in this endpoint
		mainResource := endpoint.Resources[len(endpoint.Resources)-1]
		modelName := mainResource.Name

		// Extract the endpoint info and set it to the corresponding model
		g.setEndpointInfo(modelName, endpoint)

		// Merge the properties form the request and response bodies into
		// the corresponding model
		g.mergeModelProperties(modelName, endpoint.RequestBody)
		g.mergeModelProperties(modelName, endpoint.ResponseBody)
	}
}

func (g *Generator) getURLPathForModels(url *url.URL) string {
	//TODO: Strip version path when versioning is supported
	return url.Path
}

func (g *Generator) mergeModelProperties(modelName string, body interface{}) {
	if body == nil {
		return
	}

	mInfo := g.getModelOrCreate(modelName)

	switch reflect.TypeOf(body).Kind() {
	case reflect.Map:
		props := body.(map[string]interface{})
		for propSpec, val := range props {
			g.mergeModelProperty(mInfo, propSpec, val)
		}
	case reflect.Array, reflect.Slice:
		// Get the first object of the array and start again
		arrayVal := reflect.ValueOf(body)
		if arrayVal.Len() == 0 {
			return
		}
		g.mergeModelProperties(modelName, arrayVal.Index(0).Interface())
	default:
		// This means either an empty response or a non resource response. Ignore it
		return
	}
}

func (g *Generator) mergeModelProperty(mInfo *modelInfo, propSpec string, propVal interface{}) {
	prop := newProperty(propSpec, propVal)

	_, found := mInfo.Properties[prop.Name]
	if found {
		// TODO: What to do now?. Either the old or the new one must have preference
		// We could check if prop.Type's are equal. If not -> log a warning
		// Right now old one has preference

	} else {
		mInfo.Properties[prop.Name] = prop
	}

	valKind := reflect.TypeOf(propVal).Kind()
	if valKind == reflect.Map || valKind == reflect.Array || valKind == reflect.Slice {
		g.mergeModelProperties(prop.Name, propVal)
	}
}

func (g *Generator) setEndpointInfo(modelName string, endpoint parser.Endpoint) {
	mInfo := g.getModelOrCreate(modelName)
	mInfo.EndpointsInfo = append(mInfo.EndpointsInfo, endpointInfo{
		Method:        endpoint.Method,
		URLPath:       g.getURLPathForModels(endpoint.URL),
		SegmentParams: extractSegmentParamsRenamingDups(endpoint.Resources),
		ResponseType:  getResponseType(endpoint.ResponseBody),
	})
}

func (g *Generator) getModelOrCreate(modelName string) *modelInfo {
	mInfo, modelExists := g.modelsInfo[modelName]
	if !modelExists {
		mInfo = newModelInfo(modelName)
		g.modelsInfo[modelName] = mInfo
	}
	return mInfo
}

func getResponseType(body interface{}) ResponseType {
	if body == nil {
		return EmptyResponse
	}
	switch reflect.TypeOf(body).Kind() {
	case reflect.Map:
		return ObjectResponse
	case reflect.Array, reflect.Slice:
		return ArrayResponse
	default:
		return EmptyResponse
	}
}

func extractSegmentParamsRenamingDups(resources []parser.Resource) []string {
	segmentParams := []string{}
	for _, r := range resources {
		//TODO: use r.Name to avoid duplicates
		segmentParams = append(segmentParams, r.Parameters...)
	}
	return segmentParams
}

// New creates a new Generator for the API and configured for the language passed.
func New(language Language, api *parser.Api, config Config) (Generator, error) {
	var gen specificGenerator
	var tplDir string

	switch language {
	case ObjC:
		//		gen = &ObjCGen{}
		tplDir = path.Join(templateDir, strings.ToLower(language.String()))
		//	case Android:
		//	case Swift:
	default:
		return Generator{}, errors.Annotate(ErrLangNotSupported, language.String())
	}

	generator := Generator{
		gen:    gen,
		api:    api,
		config: config,
		tplDir: tplDir,
	}

	return generator, nil
}
