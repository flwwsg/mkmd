//main
//date: 2018/5/11
//author: wdj
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

// DocTemplate template of md
const DocTemplate = `
{{if ne .APIType ""}}
** {{.APIType | printActionType}} **

字段|类型|默认值|描述|
---|---|---|---
{{end -}}
{{- .Name}} | {{.ValueType | printf "%s" }} | {{.Default }} | {{ .Desc | printDesc -}}
`

type structType struct {
	name    string
	node    *ast.StructType
	srcName string
}

// APIStruct describe api
type APIStruct struct {
	PKGName    string
	ActionID   string
	Container  *APIContainer
	ActionDesc string
}

// APIContainer accept inner struct
type APIContainer struct {
	Main []*APIDesc
	Sub  map[string]*APIContainer
}

// APIDesc get api's name, type , desc
type APIDesc struct {
	Name      string
	Alias     string
	Default   interface{}
	ValueType string
	Desc      string
	APIType   string
}

// APIDoc make markdown
type APIDoc struct {
	ActionID   string
	ActionDesc string
	Name       string
	Default    string
	ValueType  string
	Desc       string
	APIType    string
	SubName    string
}

var i = flag.String("in", ".", "api directory to generate md file")
var o = flag.String("out", "", "directory to save md file")

// var

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
	pkgs := FindPackage(input)
	d, err := os.Stat(out)
	if os.IsNotExist(err) {
		// directory not exists
		os.Mkdir(out, 0777)
	} else if err == nil && !d.IsDir() {
		// out is not directory
		flag.Usage()
		os.Exit(1)
	}
	fmt.Printf("%q", pkgs)
	ch := make(chan struct{})
	for name, pkg := range pkgs {
		go func(name string, pkg *[]*APIStruct) {
			savePath := filepath.Join(out, name+".md")
			// truncate file if savePath exists else create new file
			file, err := os.OpenFile(savePath, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0777)
			if err != nil {
				log.Fatal(err)
			}
			defer file.Close()
			for _, api := range *pkg {
				contents := FormatAPI(api)
				_, err = file.Write(contents.Bytes())
				if err != nil {
					log.Fatal(err)
				}
			}
			file.Sync()
			ch <- struct{}{}
		}(name, &pkg)
	}

	for range pkgs {
		<-ch
	}
}

// FormatAPI generate request md file
func FormatAPI(pkg *APIStruct) *bytes.Buffer {
	var printActionType = func(apiType string) string {
		if apiType == "req" {
			return "请求"
		} else if apiType == "resp" {
			return "响应"
		}
		panic(fmt.Sprintf("%s type does not supported", apiType))
	}
	var printDesc = func(desc string) string {
		if desc == "" {
			return "无"
		}
		return desc
	}
	doc, err := template.New("request").Funcs(template.FuncMap{
		"printActionType": printActionType, "printDesc": printDesc}).
		Parse(DocTemplate)
	if err != nil {
		log.Fatal(err)
	}
	var b bytes.Buffer
	role := new(APIDesc)
	role.Name = "rid"
	role.Default = "无"
	role.ValueType = "string"
	role.APIType = "req"
	pkg.Container.Main = append(pkg.Container.Main, role)
	s := "\n\n## %s %s"
	b.WriteString(fmt.Sprintf(s, pkg.ActionID, pkg.ActionDesc))
	ParseField(pkg.Container, doc, "req", &b, true, "")
	ParseField(pkg.Container, doc, "resp", &b, true, "")
	return &b
}

// ParseField get each field of api
func ParseField(c *APIContainer, doc *template.Template, apiType string, b *bytes.Buffer, addTag bool, fieldName string) {
	for _, api := range c.Main {
		if api.APIType != apiType {
			continue
		}
		newField := new(APIDoc)
		// alias works only in request
		if api.Alias != "" && apiType == "req" {
			newField.Name = api.Alias
		} else {
			newField.Name = api.Name
		}
		switch t := api.Default.(type) {
		case string:
			if api.Default.(string) == "" {
				newField.Default = "无"
			} else {
				newField.Default = api.Default.(string)
			}

		case int, int8, int16, int32, int64:
			newField.Default = fmt.Sprintf("%d", api.Default)
		case float32, float64:
			newField.Default = fmt.Sprintf("%f", api.Default)
		case bool:
			newField.Default = fmt.Sprintf("%b", api.Default)
		case nil:
			newField.Default = "无"
		default:
			panic(fmt.Sprintf("%s does not support yet", t))
		}
		newField.ValueType = api.ValueType
		newField.Desc = api.Desc
		if _, ok := c.Sub[api.ValueType]; ok {
			newField.SubName = api.ValueType
		}
		if addTag {
			newField.APIType = apiType
			addTag = false
		}
		if fieldName != "" {
			s := "\n\n** %s **\n\n字段|类型|默认值|描述|\n---|---|---|---"
			b.WriteString(fmt.Sprintf(s, fieldName))
			fieldName = ""
		}
		err := doc.Execute(b, newField)
		if err != nil {
			log.Fatal(err)
		}
	}
	for name, sub := range c.Sub {
		ParseField(sub, doc, apiType, b, false, name)
	}

}

// FindPackage get all package with specified path
func FindPackage(pkgRoot string) map[string][]*APIStruct {
	dirs := ListDir(pkgRoot, true, true)
	resp := make(map[string][]*APIStruct, len(dirs))
	for _, dir := range dirs {
		files := ListDir(dir, true, false)
		for _, file := range files {
			pkgName, actionID, pkg := getAllStruct(file)
			if len(pkg) < 1 {
				continue
			}
			t := getSingleAction(actionID, pkg)
			api := new(APIStruct)
			api.SetActionID(actionID)
			api.Container = t
			api.PKGName = pkgName
			resp[pkgName] = append(resp[pkgName], api)
		}
	}
	return resp
}

// ParseTag get api field
func (api *APIContainer) ParseTag(f *ast.Field, t string, typeName string) {
	// t = "dcapi: \"xx; xx:xxx; ddd\""
	if !api.IsValidTag(t) {
		return
	}
	t = t[strings.Index(t, "\"")+1 : strings.LastIndex(t, "\"")]
	fields := strings.Split(t, ";")
	desc := new(APIDesc)
	if strings.Contains(t, "req") {
		desc.APIType = "req"
	} else if strings.Contains(t, "resp") {
		desc.APIType = "resp"
	} else {
		// not such type
		return
	}
	desc.ValueType = typeName
	desc.Name = f.Names[0].Name
	for _, field := range fields {
		field = strings.TrimSpace(field)
		tag := strings.Split(field, ":")
		switch tag[0] {
		case "alias":
			if len(tag) > 1 {
				desc.Alias = tag[1]
			}
		case "desc":
			if len(tag) > 1 {
				desc.Desc = tag[1]
			}
		case "def":
			if len(tag) > 1 {
				desc.Default = tag[1]
			}
		}
	}
	api.Main = append(api.Main, desc)
}

// IsValidTag check tag is valid or not
func (api *APIContainer) IsValidTag(t string) bool {
	if strings.Contains(t, "skip") {
		return false
	}
	if strings.Contains(t, "-") {
		return false
	}
	return true
}

// SetActionID get action id if exists
func (api *APIStruct) SetActionID(name string) {
	re := regexp.MustCompile("[0-9]+")
	res := re.FindAllString(name, -1)
	if len(res) == 1 {
		api.ActionID = res[0]
	}
}

// helper function

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

// IsActionID check given name is action id or not
func IsActionID(name string) bool {
	re := regexp.MustCompile("[0-9]+")
	res := re.FindAllString(name, -1)
	if len(res) == 1 {
		return true
	}
	return false
}

// getAllStruct get all struct with specified file path
func getAllStruct(filePath string) (pkgName string, actionID string, allStruct []*structType) {
	if !strings.HasSuffix(filePath, "go") {
		return "", "", *new([]*structType)
	}
	fs := token.NewFileSet()
	f, err := parser.ParseFile(fs, filePath, nil, parser.ParseComments)
	if err != nil {
		log.Fatal(err)
	}
	allStruct = make([]*structType, 0)
	actionID = ""
	var findStruct = func(n ast.Node) bool {
		var structName string
		var t ast.Expr
		// get type specification
		switch x := n.(type) {
		case *ast.TypeSpec:
			structName = x.Name.Name
			if ok := IsActionID(structName); ok {
				actionID = structName
			}
			// fmt.Printf("\n==============\n%s \n", structName)
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
		s.node = x
		s.srcName = filePath
		allStruct = append(allStruct, s)
		return true
	}
	ast.Inspect(f, findStruct)
	return f.Name.String(), actionID, allStruct
}

func getSingleAction(name string, structs []*structType) *APIContainer {
	index := indexStruct(name, structs)
	if index == -1 {
		return nil
	}
	api := new(APIContainer)
	for _, f := range structs[index].node.Fields.List {
		if f.Tag == nil || !strings.Contains(f.Tag.Value, TokenTag) {
			continue
		}
		dctag := GetTag(f.Tag.Value, TokenTag)
		b, _ := ioutil.ReadFile(structs[index].srcName)
		typeName := string(b)[f.Type.Pos()-1 : f.Type.End()-1]
		api.ParseTag(f, dctag, typeName)
		c := getSingleAction(typeName, structs)
		if c != nil {
			if api.Sub == nil {
				api.Sub = make(map[string]*APIContainer, 0)
			}
			//research struct
			index = indexStruct(typeName, structs)
			api.Sub[structs[index].name] = c
		}
	}
	return api
}

// indexStruct get the index of specified name in given structs
func indexStruct(name string, s []*structType) int {
	for i, t := range s {
		if strings.Contains(name, t.name) {
			return i
		}
	}
	return -1
}
