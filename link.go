package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
)

// TokenTag tag name
const TokenTag = "dcapi"

// APITemplate template of md
const APITemplate = `
## {{.ActionID}} {{.ActionDesc}}

** 请求 **

字段|类型|默认值|描述|
---|---|---|---
{{range $i, $f := .Fields}}
{{- if eq $f.APIType "req"}}
{{- $f.Alias}} | {{$f.ValueType | printf "%s" }} | {{printDefault  $f.Default}} | {{printDesc $f.Desc }}
{{end -}}
{{end -}}

{{range $i, $typ := .ReqTypes}}

** {{$typ.Name}} **

字段|类型|默认值|描述|
---|---|---|---
{{range $i, $f := $typ.Fields}}
{{- if eq $f.APIType "req"}}
{{- $f.Alias}} | {{$f.ValueType | printf "%s" }} | {{printDefault $f.Default}} | {{ printDesc $f.Desc }}
{{end -}}
{{end -}}
{{end}}

** 响应 **

字段|类型|默认值|描述|
---|---|---|---
{{range $i, $f := .Fields}}
{{- if eq $f.APIType "resp"}}
{{- $f.Alias}} | {{$f.ValueType | printf "%s" }} | {{printDefault  $f.Default}} | {{printDesc $f.Desc }}
{{end -}}
{{end -}}
{{range $i, $typ := .RespTypes}}

** {{$typ.Name}} **

字段|类型|默认值|描述|
---|---|---|---
{{range $i, $f := $typ.Fields}}
{{- if eq $f.APIType "resp"}}
{{- $f.Alias}} | {{$f.ValueType | printf "%s" }} | {{printDefault $f.Default}} | {{ printDesc $f.Desc }}
{{end -}}
{{end -}}
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
	ReqTypes  []*StructType
	RespTypes []*StructType
	Fields    *[]*APIField
}

// StructType recording struct
type StructType struct {
	Name   string
	Fields []*APIField
}

var i = flag.String("in", ".", "api directory to generate md file")
var o = flag.String("out", "", "directory to save md file")

func main() {
	// find file path
	flag.Parse()
	if *i == "" || *o == "" {
		flag.Usage()
		os.Exit(1)
	}
	out, err := filepath.Abs(*o)
	if err != nil {
		log.Fatal(err)
	}
	input, err := filepath.Abs(*i)
	if err != nil {
		log.Fatal(err)
	}
	pkgs := pkgStructs(input)
	d, err := os.Stat(out)
	if os.IsNotExist(err) {
		// directory not exists
		os.Mkdir(out, 0777)
	} else if err == nil && !d.IsDir() {
		// out is not directory
		flag.Usage()
		os.Exit(1)
	}
	ch := make(chan struct{})
	for name, pkg := range pkgs {
		go func(name string, pkg map[string]*StructType) {
			savePath := filepath.Join(out, name+".md")
			// truncate file if savePath exists else create new file
			file, err := os.OpenFile(savePath, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0777)
			if err != nil {
				log.Fatal(err)
			}
			defer file.Close()
			acts := GenAPI(&pkg)
			for _, act := range acts {
				b := FormatSingleAPI(act)
				//fmt.Println(b.String())
				_, err = file.Write(b.Bytes())
				if err != nil {
					log.Fatal(err)
				}
			}
			file.Sync()
			ch <- struct{}{}
		}(name, pkg)
	}

	for range pkgs {
		<-ch
	}
}

func (s *StructType) isActionID() bool {
	re := regexp.MustCompile("[0-9]+")
	res := re.FindAllString(s.Name, -1)
	if len(res) == 1 {
		return true
	}
	return false
}

func (s *StructType) isType(typeName string) bool {
	if strings.Contains(typeName, s.Name) {
		return true
	}
	return false
}

//SetActionID get action id from specified action Name
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
		if f == "" {
			continue
		}
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
		case "req", "resp":
		default:
			panic(fmt.Sprintf("tag %s does not supported", tag[0]))
		}
	}
	if field.Alias == "" {
		field.Alias = field.Name
	}
}

// FormatSingleAPI generate a single api markdown file
func FormatSingleAPI(api *SingleAPI) *bytes.Buffer {
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
	if len(*api.Fields) == 0 {
		t := &APIField{"无", "无", "无", "无", "无", "req"}
		*api.Fields = append(*api.Fields, t)
	}
	doc, err := template.New("request").Funcs(template.FuncMap{"printDesc": printDesc, "printDefault": printDefault}).
		Parse(APITemplate)
	if err != nil {
		log.Fatal(err)
	}
	b := new(bytes.Buffer)
	doc.Execute(b, api)
	//fmt.Print(b.String())
	return b
}

//GenAPI generating single api with actionID
func GenAPI(pkg *map[string]*StructType) []*SingleAPI {
	apiList := make([]*SingleAPI, 0)
	for name, st := range *pkg {
		if st.isActionID() {
			api := new(SingleAPI)
			api.SetActionID(name)
			api.Fields = &st.Fields
			for _, field := range *api.Fields {
				GetCustomTypes(api, field, pkg)
			}
			apiList = append(apiList, api)
		}
	}
	return apiList
}

// GetCustomTypes find custom struct type in api using deep search
func GetCustomTypes(api *SingleAPI, field *APIField, pkg *map[string]*StructType) {
	typeName := field.ValueType
	if s := findTypeStruct(typeName, pkg); s != nil {
		if field.APIType == "req" {
			api.ReqTypes = append(api.ReqTypes, s)
		}
		if field.APIType == "resp" {
			api.RespTypes = append(api.RespTypes, s)
		}
		for _, field := range s.Fields {
			GetCustomTypes(api, field, pkg)
		}
	}
}

// pkgStructs collect struct from giving package path
func pkgStructs(srcPath string) map[string]map[string]*StructType {
	dirs := ListDir(srcPath, true, true)
	resp := make(map[string]map[string]*StructType, len(dirs))
	for _, dir := range dirs {
		files := ListDir(dir, true, false)
		for _, file := range files {
			pkgName, pkg := collectStructs(file)
			if len(pkg) < 1 {
				continue
			}
			if resp[pkgName] == nil {
				resp[pkgName] = make(map[string]*StructType)
			}
			for k, v := range pkg {
				resp[pkgName][k] = v
			}
		}
	}
	return resp
}

func collectStructs(srcPath string) (string, map[string]*StructType) {
	allStruct := make(map[string]*StructType, 0)
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
		s := new(StructType)
		s.Name = structName
		s.Fields = genField(x, srcPath)
		allStruct[structName] = s
		return true
	}
	ast.Inspect(f, findStruct)
	return f.Name.String(), allStruct
}

func genField(node *ast.StructType, srcPath string) []*APIField {
	b, _ := ioutil.ReadFile(srcPath)
	field := make([]*APIField, 0)
	for _, f := range node.Fields.List {
		newField := new(APIField)
		//ignore invalid tag
		if f.Tag == nil || !strings.Contains(f.Tag.Value, TokenTag) {
			continue
		}
		dctag := GetTag(f.Tag.Value, TokenTag)
		typeName := string(b)[f.Type.Pos()-1 : f.Type.End()-1]
		newField.ParseTag(f, dctag, typeName)
		field = append(field, newField)
	}
	return field
}

//helper function

func findTypeStruct(name string, pkg *map[string]*StructType) *StructType {
	for _, s := range *pkg {
		if s.isType(name) {
			return s
		}
	}
	return nil
}

// ListDir list directories and files in fpath
func ListDir(fpath string, fullPath bool, listDir bool) []string {
	files, err := ioutil.ReadDir(fpath)
	dirs := make([]string, 0)
	fileName := ""
	if err != nil {
		log.Printf("list error path %s", fpath)
		log.Fatal(err)
	}
	for _, f := range files {
		if fullPath {
			fileName = path.Join(fpath, f.Name())
		} else {
			fileName = f.Name()
		}
		if f.IsDir() == listDir {
			dirs = append(dirs, fileName)
		}
	}
	return dirs
}

// IsActionID check given name is action id or not
func IsActionID(name string) bool {
	re := regexp.MustCompile("[0-9]+")
	res := re.FindAllString(name, -1)
	if len(res) == 1 {
		return true
	}
	return false
}

// GetTag find tag with specified token
func GetTag(t string, tk string) string {
	// tag = "`dcapi:"ass:xxx; sss""
	dcStart := strings.Index(t, tk)
	firstQ := strings.Index(t[dcStart:], `"`)
	dcEnd := strings.Index(t[dcStart+firstQ+1:], `"`)
	if dcEnd != -1 && dcStart != -1 {
		return t[dcStart : dcStart+firstQ+dcEnd+2]
	}
	return ""
}
