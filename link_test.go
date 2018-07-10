package main

import "testing"

//var pkgRoot = "."

func TestFindActionID(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name  string
		args  args
		want  string
		want1 bool
	}{
		{
			"IgnoreTestFile", args{"demo_1999_test.go"}, "", false,
		},
		{
			"IgnoreTestFile", args{"demo_1999_test"}, "", false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := FindActionID(tt.args.s)
			if got != tt.want {
				t.Errorf("FindActionID() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("FindActionID() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}
