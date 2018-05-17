package main

import (
	"fmt"
	"testing"
)

var pkgPath = "."

func TestCollectStruct(t *testing.T) {
	pkgs := pkgStructs(pkgPath)
	pkg, ok := pkgs["pkg1"]
	if !ok {
		t.Error("pkg1 does not collected")
	}
	s, ok := pkg["Demo999"]
	if !ok {
		t.Error("Demo999 does not collected in pkg1")
	}
	for _, f := range s.Fields {
		if f.Name == "" {
			t.Errorf("Demo99 does not has enough field:  %s", s.Fields)
		}
	}
	pkg, ok = pkgs["pkg2"]
	if !ok {
		t.Error("pkg2 does not collected")
	}
	pkg, ok = pkgs["pkg3"]
	if !ok {
		t.Error("pkg3 does not collected")
	}
	fmt.Printf("%v", pkg)
}

func TestGenAPI(t *testing.T) {
	pkgs := pkgStructs(pkgPath)
	pkg, _ := pkgs["pkg1"]
	acts := GenAPI(&pkg)
	if len(acts) != 1 {
		t.Error("pkg1 does not have action api")
	}
	if len(acts[0].ReqTypes) != 0 {
		t.Error("pkg1 does not have action custome struct info")
	}
	pkg, _ = pkgs["pkg3"]
	acts = GenAPI(&pkg)
	for _, act := range acts {
		if act.ActionID == "2000" {
			if len(act.RespTypes) != 2 {
				t.Error("actionID 2000 does have 2 custome struct info")
			}
		}
	}
	pkg, _ = pkgs["pkg2"]
	acts = GenAPI(&pkg)
	for _, act := range acts {
		if len(*act.Fields) != 4 {
			fmt.Printf("%q\n", act.Fields)
			t.Error("error fields")
		}
	}

}

func TestFormatSingleAPI(t *testing.T) {
	pkgs := pkgStructs(pkgPath)
	pkg, _ := pkgs["pkg3"]
	acts := GenAPI(&pkg)
	fmt.Printf("%q\n", acts[0].Fields)
	FormatSingleAPI(acts[0])
	FormatSingleAPI(acts[1])
	t.Fail()
}
