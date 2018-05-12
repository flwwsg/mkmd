package main

import (
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
	if resp.ActionID != "999" {
		t.Errorf("demo actionID %s != 999", resp.ActionID)
	}
	if len(resp.Req) != 1 {
		t.Error("api demo does not has request")
	}
	if len(resp.Resp) != 1 {
		t.Error("api demo does not has resp")
	}
	for _, req := range resp.Req {
		if req.Name != "ID" && req.Alias != "id" {
			t.Error("api demo does not has id request")
		}
	}

	for _, r := range resp.Resp {
		if r.Name != "Number" {
			t.Error("api demo does not has Number resp")
		}
	}

}

// find req tag
func TestFindREQTag(t *testing.T) {
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
	if pkg.Req[0].Alias != "fid" {
		t.Error("pkg3 does not has fid")
	}
	t.Errorf("%q", pkg)

}

// format md
