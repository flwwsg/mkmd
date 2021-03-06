package main

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"log"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"text/template"
)

// TokenTag tag name
const TokenTag = "valid"

// APITemplate template of md
const APITemplate = `
### {{.ActionID}} {{.ActionDesc}}

#### 请求

字段|类型|描述|
---|---|---
{{range $i, $f := .ReqFields}}
{{- $f.Alias}} | {{$f.ValueType | printf "%s" }}| {{printDesc $f.Desc }}
{{end}}

#### 响应

字段|类型|描述|
---|---|---
{{range $i, $f := .RespFields}}
{{- $f.Alias}} | {{$f.ValueType | printf "%s" }} | {{printDesc $f.Desc }}
{{end -}}
`

// CustomTypeTemplate template for custom type
const CustomTypeTemplate = `
### 自定义数据类型
{{$length := len .}}
{{- if eq $length 0}}
#### 无
{{end -}}
{{range $i, $typ := .}}
#### {{$typ.Name}}
字段|类型|描述|
---|---|---
{{range $i, $f := $typ.Fields}}
{{- $f.Alias}} | {{$f.ValueType | printf "%s" }} | {{printDesc $f.Desc }}
{{end -}}
{{end -}}
`

// APIField each field of an api
type APIField struct {
	Name string
	//display name
	Alias     string
	ValueType string
	Desc      string
	Required  bool
}

// SingleAPI an api
type SingleAPI struct {
	ActionID   string
	ActionDesc string
	ReqFields  *[]*APIField
	RespFields *[]*APIField
}

// ReqAPI request part of api
type ReqAPI struct {
	CustomTypes []*StructType
	ActionID    string
	ActionDesc  string
	Fields      *[]*APIField
}

// RespAPI resp part of api
type RespAPI struct {
	CustomTypes []*StructType
	ActionID    string
	ActionDesc  string
	Fields      *[]*APIField
}

// StructType recording struct
type StructType struct {
	Name     string
	ActionID string //mark action id to specify API, only request struct will be marked
	Fields   []*APIField
	Desc     string
}

// CustomTypes diy type
type CustomTypes interface {
	AddTypes(structType *StructType)
}

//var i = flag.String("in", "", "api directory to generate md file")

func main() {
	////find file path
	//flag.Parse()
	//if *i == "" {
	//	flag.Usage()
	//	os.Exit(1)
	//}
	println(GenDoc("./pkg2"))
}

//GenDoc generating api file
func GenDoc(apiPath string) string {
	input, err := filepath.Abs(apiPath)
	if err != nil {
		log.Fatal(err)
	}
	_, structs := pkgStructs(input)
	req, resp := GenAPI(&structs)
	rtn := make([]string, len(resp)+1)
	idx := make([]int, len(resp))
	for i, api := range resp {
		n, err := strconv.Atoi(api.ActionID)
		if err != nil {
			log.Fatal(err)
		}
		idx[i] = n
	}
	sort.Ints(idx)
	customTypes := make([]*StructType, 0)
	for i, aid := range idx {
		strAID := strconv.Itoa(aid)
		for _, respAPI := range resp {
			if respAPI.ActionID == strAID {
				//find request struct
				customTypes = append(customTypes, respAPI.CustomTypes...)
				for _, reqAPI := range req {
					if reqAPI.ActionID == strAID {
						customTypes = append(customTypes, reqAPI.CustomTypes...)
						b := FormatSingleAPI(reqAPI, respAPI)
						rtn[i+1] = b.String()
						break
					}
				}
			}
		}
	}
	customTypes = unique(customTypes)
	b := FormatCustomTypes(customTypes)
	rtn[0] = b.String()
	s := strings.Join(rtn, "")
	return s
}

func (s *StructType) isReq() bool {
	//request struct name is like DemoLoginParams
	l := len(s.Name)
	if l < 6 {
		return false
	}
	if s.Name[l-6:] == "Params" {
		return true
	}
	return false
}

func (s *StructType) isResp() bool {
	//response struct name is like DemoLoginResp
	l := len(s.Name)
	if l < 4 {
		return false
	}
	if s.Name[l-4:] == "Resp" {
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

func (s *StructType) SetDesc(comm string) {
	//drop struct name
	desc := strings.Replace(comm, s.Name, "", 1)
	desc = strings.Replace(desc, "\n", " ", -1)
	s.Desc = strings.TrimSpace(desc)
}

//AddTypes append custom types
func (req *ReqAPI) AddTypes(s *StructType) {
	req.CustomTypes = append(req.CustomTypes, s)
}

// AddTypes add custom types
func (resp *RespAPI) AddTypes(s *StructType) {
	resp.CustomTypes = append(resp.CustomTypes, s)
}

// IsValidTag check tag is valid or not
func (field *APIField) IsValidTag(t string) bool {
	if strings.Contains(t, "-") {
		return false
	}
	return true
}

func (field *APIField) SetDesc(s string) {
	desc := strings.Replace(s, field.Name, "", 1)
	desc = strings.Replace(desc, "\n", " ", -1)
	field.Desc = strings.TrimSpace(desc)

}

//ParseTag handle tag
func (field *APIField) ParseTag(f *ast.Field, t string) {
	// t = "valid: \"required, xxx\""
	if !field.IsValidTag(t) {
		return
	}
	t = t[strings.Index(t, "\"")+1 : strings.LastIndex(t, "\"")]
	fields := strings.Split(t, ",")
	field.Required = false
	for _, f := range fields {
		f = strings.TrimSpace(f)
		if f == "" {
			continue
		}
		switch f {
		case "required":
			field.Required = true
		case "optional":
		default:
			continue
		}
	}
}

// FormatSingleAPI generate a single api markdown file
func FormatSingleAPI(req *ReqAPI, resp *RespAPI) *bytes.Buffer {
	var printDesc = func(desc string) string {
		if desc == "" {
			return "无"
		}
		return strings.TrimSpace(desc)
	}
	var printNeed = func(need bool) string {
		if need {
			return "是"
		}
		return "否"

	}
	api := new(SingleAPI)
	api.ActionID = resp.ActionID
	api.ActionDesc = resp.ActionDesc
	api.RespFields = resp.Fields
	if req.ActionDesc != "" {
		api.ActionDesc = req.ActionDesc
	}
	api.ReqFields = req.Fields
	doc, err := template.New("request").Funcs(template.FuncMap{"printDesc": printDesc, "printNeed": printNeed}).
		Parse(APITemplate)
	if err != nil {
		log.Fatal(err)
	}
	b := new(bytes.Buffer)
	doc.Execute(b, api)
	return b
}

// FormatCustomTypes format cutom types in template
func FormatCustomTypes(api []*StructType) *bytes.Buffer {
	var printDesc = func(desc string) string {
		if desc == "" {
			return "无"
		}
		return strings.TrimSpace(desc)
	}
	doc, err := template.New("customTypes").Funcs(template.FuncMap{"printDesc": printDesc}).
		Parse(CustomTypeTemplate)
	if err != nil {
		log.Fatal(err)
	}
	b := new(bytes.Buffer)
	doc.Execute(b, api)
	return b
}

//GenAPI generating single api with actionID
func GenAPI(pkg *map[string]*StructType) ([]*ReqAPI, []*RespAPI) {
	reqs := make([]*ReqAPI, 0)
	resps := make([]*RespAPI, 0)
	for _, st := range *pkg {
		if st.isReq() {
			//get request struct
			req := new(ReqAPI)
			req.ActionID = st.ActionID
			req.ActionDesc = st.Desc
			req.Fields = &st.Fields
			for _, field := range *req.Fields {
				GetCustomTypes(req, field, pkg)
			}
			reqs = append(reqs, req)
			//fmt.Printf("request struct is %v:\n", req)
		}
		if st.isResp() {
			//get resp struct
			resp := new(RespAPI)
			resp.ActionID = st.ActionID
			resp.ActionDesc = st.Desc
			resp.Fields = &st.Fields
			for _, field := range *resp.Fields {
				GetCustomTypes(resp, field, pkg)
			}
			resps = append(resps, resp)
			//fmt.Printf("response struct is %v:\n", resp)
		}
	}
	return reqs, resps
}

// GetCustomTypes find custom struct type in api using deep search
func GetCustomTypes(api CustomTypes, field *APIField, pkg *map[string]*StructType) {
	typeName := field.ValueType
	if s := findTypeStruct(typeName, pkg); s != nil {
		api.AddTypes(s)
		for _, field := range s.Fields {
			GetCustomTypes(api, field, pkg)
		}
	}
}

// pkgStructs collect all struct from giving package path
func pkgStructs(pkgPath string) (string, map[string]*StructType) {
	resp := make(map[string]*StructType)
	files := ListDir(pkgPath, true, false)
	pkgName := ""
	for _, file := range files {
		getReq := false
		getResp := false
		fileName := filepath.Base(file)
		actionID, ok := FindActionID(fileName)
		if !ok {
			continue
		}
		//get package name and map[structName] struct
		name, structs := collectStructs(file)
		for k, v := range structs {
			// add empty filed in struct
			if len(v.Fields) == 0 {
				v.Fields = []*APIField{emptyField()}
			}
			if v.isReq() {
				v.ActionID = actionID
				getReq = true
			}
			if v.isResp() {
				v.ActionID = actionID
				getResp = true
			}
			resp[k] = v
		}
		if !getReq {
			defaultReq := defaultReq(actionID)
			resp[defaultReq.Name] = defaultReq
		}
		if !getResp {
			defaultResp := defaultResp(actionID)
			resp[defaultResp.Name] = defaultResp
		}
		pkgName = name
	}
	return pkgName, resp
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
		var structDec string
		// get type specification
		switch x := n.(type) {
		case *ast.GenDecl:
			if len(x.Specs) != 1 {
				return true
			}
			structDec = x.Doc.Text()
			switch xs := x.Specs[0].(type) {
			case *ast.TypeSpec:
				structName = xs.Name.Name
				t = xs.Type
			}
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
		s.SetDesc(structDec)
		allStruct[structName] = s
		return true
	}
	ast.Inspect(f, findStruct)
	//ast.Print(fs, f)
	return f.Name.String(), allStruct
}

func genField(node *ast.StructType, srcPath string) []*APIField {
	b, _ := ioutil.ReadFile(srcPath)
	field := make([]*APIField, 0)
	for _, f := range node.Fields.List {
		newField := new(APIField)
		//ignore invalid tag
		if f.Tag != nil && strings.Contains(f.Tag.Value, TokenTag) {
			tags := GetTag(f.Tag.Value, TokenTag)
			newField.ParseTag(f, tags)
		}
		typeName := string(b)[f.Type.Pos()-1 : f.Type.End()-1]
		newField.ValueType = typeName
		newField.Name = f.Names[0].Name
		if f.Comment.Text() != "" {
			newField.SetDesc(f.Comment.Text())
		} else {
			newField.SetDesc(f.Doc.Text())
		}
		newField.Alias = newField.Name
		field = append(field, newField)
	}
	return field
}

//helper function

func emptyField() *APIField {
	field := new(APIField)
	field.Name = "无"
	field.Alias = field.Name
	field.ValueType = "无"
	field.Desc = ""
	field.Required = false
	return field
}

//default request
func defaultReq(aid string) *StructType {
	s := new(StructType)
	s.ActionID = aid
	s.Name = "Default" + aid + "Params"
	//roleID
	rid := new(APIField)
	rid.Name = "RoleId"
	rid.Alias = rid.Name
	rid.ValueType = "string"
	rid.Required = true
	rid.Desc = "角色id"
	s.Fields = []*APIField{rid}
	return s
}

func defaultResp(aid string) *StructType {
	s := new(StructType)
	s.ActionID = aid
	s.Name = "Default" + aid + "Resp"
	info := new(APIField)
	info.Name = "Info"
	info.Alias = info.Name
	info.ValueType = "string"
	code := new(APIField)
	code.Name = "Code"
	code.Alias = code.Name
	code.ValueType = "int"
	code.Desc = "0 成功， 1 失败"
	s.Fields = []*APIField{info, code}
	return s
}

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

// GetTag find tag with specified token
func GetTag(t string, tk string) string {
	// tag = "`valid:"ass:xxx; sss""
	tagStart := strings.Index(t, tk)
	firstQ := strings.Index(t[tagStart:], `"`)
	tagEnd := strings.Index(t[tagStart+firstQ+1:], `"`)
	if tagEnd != -1 && tagStart != -1 {
		return t[tagStart : tagStart+firstQ+tagEnd+2]
	}
	return ""
}

func unique(l []*StructType) []*StructType {
	keys := make(map[string]bool)
	newList := make([]*StructType, 0)
	for _, s := range l {
		if _, v := keys[s.Name]; !v {
			keys[s.Name] = true
			newList = append(newList, s)
		}
	}
	return newList
}

//FindActionID if find actionID, return actionID and identifier(bool)
func FindActionID(s string) (string, bool) {
	t := strings.Split(s, "_")
	if t[len(t)-1] == "test" || t[len(t)-1] == "test.go" {
		return "", false
	}
	re := regexp.MustCompile("[0-9]+")
	res := re.FindAllString(s, -1)
	if len(res) == 1 {
		return res[0], true
	}
	return "", false
}
