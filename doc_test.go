package main

import (
	"fmt"
	"testing"
)

var pkgPath = `/home/lblue/dev/go/src/mkmd`

// find all package
func TestFindPackage(t *testing.T) {
	pkgs := FindPackage(pkgPath)
	resp, err := pkgs["pkg1"]
	if !err {
		t.Error("can not find package `pkg1`")
	}
	if resp[0].ActionID != "999" {
		t.Errorf("demo actionID %s != 999", resp[0].ActionID)
	}
	if len((*resp[0].Container).Main) != 2 {
		t.Error("api demo does not has request")
	}
	for _, field := range (*resp[0].Container).Main {
		switch field.Name {
		case "ID":
			if field.Alias != "id" {
				t.Error("api demo does not has id field")
			}
		case "Number":
		default:
			t.Error("api demo does not has Number or ID field")
		}
	}
}

// find multiple package
func TestFindMuliplePackage(t *testing.T) {
	pkgs := FindPackage(pkgPath)
	if _, ok := pkgs["pkg1"]; !ok {
		t.Error("pkg1 does not in package")
	}
	if _, ok := pkgs["pkg2"]; !ok {
		t.Error("pkg2 does not in package")
	}
}

// find inner struct
func TestInnerStruct(t *testing.T) {
	pkg, ok := FindPackage(pkgPath)["pkg3"]
	if !ok {
		t.Error("pkg3 does not in package")
	}
	for _, field := range (*pkg[0].Container).Main {
		switch field.Name {
		case "FamilyID":
			if field.Alias != "fid" {
				t.Error("api demo does not has fid field")
			}
		case "ChildInfo":
			tname := fmt.Sprintf("%s", field.ValueType)
			fmt.Print(tname)
			resp, ok := (*pkg[0].Container).Sub[tname]
			if !ok {
				t.Errorf("api demo does not find inner struct %s", tname)
			}
			if resp.Sub == nil {
				t.Errorf("api's inner struct %s has bad inner struct", tname)
			}
		case "FID":
		case "MiracleTrigger":
		case "Demo":
		default:
			t.Errorf("api demo has bad field %s", field.Name)
		}
	}
}

// format md

func TestFormateAPI(t *testing.T) {
	pkg, ok := FindPackage(pkgPath)["family"]
	if !ok {
		t.Error("api demo does not find pkg1")
	}
	txt := FormatAPI(pkg[0])
	expected := `
** 请求 **
字段|类型|默认值|描述|
---|---|---|---
ID|string|-|-|
	`
	if txt.String() != expected {
		t.Error("request md is ", txt)
	}

}
