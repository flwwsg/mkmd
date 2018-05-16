package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"log"
	"regexp"
	"strings"
	"text/template"
)

// APITemplate template of md
const APITemplate = `
## {{.ActionID}} {{.ActionDesc}}

** {{.APIType}} **

字段|类型|默认值|描述|
---|---|---|---
{{range $i, $f := .Fields}}
{{- $f.Name}} | {{$f.ValueType | printf "%s" }} | {{$f.Default | printDefault }} | {{ $f.Desc | printDesc -}}
{{end}}

{{range $i, $typ := .Types}}

** {{$typ.name}} **

字段|类型|默认值|描述|
---|---|---|---
{{range $i, $f := $typ.fields}}
{{- $f.Name}} | {{$f.ValueType | printf "%s" }} | {{$f.Default | printDefault }} | {{ $f.Desc | printDesc -}}
{{end}}
{{end}}
`

// APIField each field of an api
type APIField struct {
	Name      string
	Alias     string
	Default   interface{}
	ValueType string
	Desc      string
	APIType   string
}

// SingleAPI an api
type SingleAPI struct {
	ActionID   string
	ActionDesc string
	// structs has being used
	Types   []*structType
	Fields  *[]*APIField
	APIType string
}

type structType struct {
	name    string
	node    *ast.StructType
	srcName string
	fields  []*APIField
}

func (s *structType) isActionID() bool {
	re := regexp.MustCompile("[0-9]+")
	res := re.FindAllString(s.name, -1)
	if len(res) == 1 {
		return true
	}
	return false
}

func (s *structType) isType(typeName string) bool {
	if strings.Contains(typeName, s.name) {
		return true
	}
	return false
}

//SetActionID get action id from specified action name
func (api *SingleAPI) SetActionID(name string) {
	re := regexp.MustCompile("[0-9]+")
	res := re.FindAllString(name, -1)
	api.ActionID = res[0]
}

// IsValidTag check tag is valid or not
func (field *APIField) IsValidTag(t string) bool {
	if strings.Contains(t, "skip") {
		return false
	}
	if strings.Contains(t, "-") {
		return false
	}
	return true
}

//ParseTag handle dctag
func (field *APIField) ParseTag(f *ast.Field, t string, typeName string) {
	// t = "dcapi: \"xx; xx:xxx; ddd\""
	if !field.IsValidTag(t) {
		return
	}
	t = t[strings.Index(t, "\"")+1 : strings.LastIndex(t, "\"")]
	fields := strings.Split(t, ";")
	if strings.Contains(t, "req") {
		field.APIType = "req"
	} else if strings.Contains(t, "resp") {
		field.APIType = "resp"
	} else {
		// not such type
		return
	}
	field.ValueType = typeName
	field.Name = f.Names[0].Name
	for _, f := range fields {
		f = strings.TrimSpace(f)
		tag := strings.Split(f, ":")
		switch tag[0] {
		case "alias":
			if len(tag) > 1 {
				field.Alias = tag[1]
			}
		case "desc":
			if len(tag) > 1 {
				field.Desc = tag[1]
			}
		case "def":
			if len(tag) > 1 {
				field.Default = tag[1]
			}
		}
	}
}

// FormatSingleAPI generat a single api markdown file
func FormatSingleAPI(api *SingleAPI) {
	var printDesc = func(desc string) string {
		if desc == "" {
			return "无"
		}
		return desc
	}
	var printDefault = func(defValue interface{}) string {
		switch t := defValue.(type) {
		case string:
			if defValue.(string) == "" {
				return "无"
			}
			return defValue.(string)
		case int, int8, int16, int32, int64:
			return fmt.Sprintf("%d", defValue)
		case float32, float64:
			return fmt.Sprintf("%f", defValue)
		case bool:
			return fmt.Sprintf("%b", defValue)
		case nil:
			return "无"
		default:
			panic(fmt.Sprintf("%s does not support yet", t))
		}
	}
	apiType := "请求"
	api.APIType = apiType
	doc, err := template.New("request").Funcs(template.FuncMap{"printDesc": printDesc, "printDefault": printDefault}).
		Parse(APITemplate)
	if err != nil {
		log.Fatal(err)
	}
	var b bytes.Buffer
	doc.Execute(&b, api)
	fmt.Print(b.String())
}

//GenAPI generating single api with actionID
func GenAPI(pkg *map[string]*structType) []*SingleAPI {
	apiList := make([]*SingleAPI, 0)
	for name, st := range *pkg {
		if st.isActionID() {
			api := new(SingleAPI)
			api.SetActionID(name)
			api.Fields = &st.fields
			for _, field := range *api.Fields {
				GetCustomTypes(api, field.ValueType, pkg)
			}
			apiList = append(apiList, api)
		}
	}
	return apiList
}

// GetCustomTypes find custom struct type in api using deep search
func GetCustomTypes(api *SingleAPI, typeName string, pkg *map[string]*structType) {
	if s := findTypeStruct(typeName, pkg); s != nil {
		api.Types = append(api.Types, s)
		for _, field := range s.fields {
			GetCustomTypes(api, field.ValueType, pkg)
		}
	}
}

// pkgStructs collect struct from giving package path
func pkgStructs(srcPath string) map[string]map[string]*structType {
	dirs := ListDir(srcPath, true, true)
	resp := make(map[string]map[string]*structType, len(dirs))
	for _, dir := range dirs {
		files := ListDir(dir, true, false)
		for _, file := range files {
			pkgName, pkg := collectStructs(file)
			if len(pkg) < 1 {
				continue
			}
			if resp[pkgName] == nil {
				resp[pkgName] = make(map[string]*structType)
			}
			for k, v := range pkg {
				resp[pkgName][k] = v
			}
		}
	}
	return resp
}

func collectStructs(srcPath string) (string, map[string]*structType) {
	allStruct := make(map[string]*structType, 0)
	if !strings.HasSuffix(srcPath, "go") {
		return "", allStruct
	}
	fs := token.NewFileSet()
	f, err := parser.ParseFile(fs, srcPath, nil, parser.ParseComments)
	if err != nil {
		log.Fatal(err)
	}
	var findStruct = func(n ast.Node) bool {
		var structName string
		var t ast.Expr
		// get type specification
		switch x := n.(type) {
		case *ast.TypeSpec:
			structName = x.Name.Name
			if ok := IsActionID(structName); ok {
			}
			t = x.Type
		default:
			return true
		}
		x, ok := t.(*ast.StructType)
		if !ok {
			return true
		}
		s := new(structType)
		s.name = structName
		s.fields = genField(x, srcPath)
		allStruct[structName] = s
		return true
	}
	ast.Inspect(f, findStruct)
	return f.Name.String(), allStruct
}

func genField(node *ast.StructType, srcPath string) []*APIField {
	b, _ := ioutil.ReadFile(srcPath)
	field := make([]*APIField, len(node.Fields.List))
	for i, f := range node.Fields.List {
		newField := new(APIField)
		//ignore invalid tag
		if f.Tag == nil || !strings.Contains(f.Tag.Value, TokenTag) {
			continue
		}
		dctag := GetTag(f.Tag.Value, TokenTag)
		typeName := string(b)[f.Type.Pos()-1 : f.Type.End()-1]
		newField.ParseTag(f, dctag, typeName)
		field[i] = newField
	}
	return field
}

//helper function

func findTypeStruct(name string, pkg *map[string]*structType) *structType {
	for _, s := range *pkg {
		if s.isType(name) {
			return s
		}
	}
	return nil
}
