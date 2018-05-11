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
	Req      []*APIDesc
	Resp     []*APIDesc
}

// APIDesc get api's name, type , desc
type APIDesc struct {
	Name      string
	Alias     string
	Default   interface{}
	APIType   ast.Expr
	Desc      string
	ValueType interface{}
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
			if !strings.HasSuffix(file, "go") {
				continue
			}
			fs := token.NewFileSet()
			f, err := parser.ParseFile(fs, file, nil, parser.ParseComments)
			if err != nil {
				log.Fatal(err)
			}
			pkgname := f.Name.String()
			api := new(APIStruct)
			api.PKGName = pkgname
			resp[pkgname] = api
			var findStruct = func(n ast.Node) bool {
				var actionName string
				var t ast.Expr
				// get type specification
				switch x := n.(type) {
				case *ast.TypeSpec:
					actionName = x.Name.Name
					t = x.Type
				default:
					return true
				}
				x, ok := t.(*ast.StructType)
				if !ok {
					return true
				}
				api.AddActionID(actionName)
				for _, f := range x.Fields.List {
					if f.Tag == nil {
						continue
					}
					dctag := GetTag(f.Tag.Value, TokenTag)
					api.ParseTag(f, dctag)
				}
				return true
			}
			ast.Inspect(f, findStruct)
		}

	}
	return resp
}

// ParseTag get api field
func (api *APIStruct) ParseTag(f *ast.Field, t string) {
	// t = "dcapi: xx; xx:xxx; ddd"
	if strings.Contains(t, "skip") {
		return
	}
	t = t[strings.Index(t, ":")+1:]
	fields := strings.Split(t, ";")
	desc := new(APIDesc)
	desc.APIType = f.Type
	if strings.Contains(t, "req") {
		api.Req = append(api.Req, desc)
	} else if strings.Contains(t, "resp") {
		api.Resp = append(api.Resp, desc)
	} else {
		// not such type
		return
	}
	for _, field := range fields {
		field = strings.TrimSpace(field)
		tag := strings.Split(field, ":")
		switch tag[0] {
		case "alias":
			if len(tag) > 1 {
				desc.Name = tag[1]
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
}

// AddActionID get action id if exists
func (api *APIStruct) AddActionID(name string) {
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
