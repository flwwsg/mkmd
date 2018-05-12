//main
//date: 2018/5/11
//author: wdj
package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"log"
	"path"
	"regexp"
	"strings"
)

// TokenTag tag name
const TokenTag = "dcapi"

type structType struct {
	name string
	node *ast.StructType
}

// APIStruct descript api
type APIStruct struct {
	PKGName  string
	ActionID string
	Main     *[]*APIDesc
	Sub      *map[string]*APIContainer
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
	ValueType ast.Expr
	Desc      string
	APIType   string
}

func main() {
	// find file path
	fs := token.NewFileSet()
	f, err := parser.ParseFile(fs, "pkg1/demo.go", nil, parser.ParseComments)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%q\n", f)
	fmt.Printf("%s\n", f.Name)
	ast.Inspect(f, findStruct)
}

func findStruct(n ast.Node) bool {
	x, ok := n.(*ast.StructType)
	if !ok {
		return true
	}
	for _, f := range x.Fields.List {
		fmt.Println("tag is ", f.Tag)
		fmt.Println("type is ", f.Type)
		fmt.Println("name is ", f.Names)
	}
	return true
}

// FindPackage get all package with specified path
func FindPackage(pkgRoot string) map[string]*APIStruct {
	dirs := ListDir(pkgRoot, true, true)
	resp := make(map[string]*APIStruct, len(dirs))
	for _, dir := range dirs {
		files := ListDir(dir, true, false)
		for _, file := range files {
			pkgName, actionID, pkg := getAllStruct(file)
			if len(pkg) < 1 {
				continue
			}
			t := getSingleAction(actionID, pkg)
			// fmt.Println("---------------------")
			// fmt.Printf("%q\n", t)
			api := new(APIStruct)
			api.SetActionID(actionID)
			api.Main = &t.Main
			api.Sub = &t.Sub
			api.PKGName = pkgName
			resp[pkgName] = api
		}
	}
	return resp
}

// ParseTag get api field
func (api *APIContainer) ParseTag(f *ast.Field, t string) {
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
	desc.ValueType = f.Type
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
			fmt.Printf("\n==============\n%s \n", structName)
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
		api.ParseTag(f, dctag)
		typeName := fmt.Sprintf("%s", f.Type)
		c := getSingleAction(typeName, structs)
		if c != nil {
			if api.Sub == nil {
				api.Sub = make(map[string]*APIContainer, 0)
			}
			api.Sub[typeName] = c
		}
	}
	return api
}

// indexStruct get the index of specified name in given structs
func indexStruct(name string, s []*structType) int {
	for i, t := range s {
		if t.name == name {
			return i
		}
	}
	return -1
}
