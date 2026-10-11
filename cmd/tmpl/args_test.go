package main

import (
	"reflect"
	"testing"
)

func TestParseArgs(t *testing.T) {
	cases := []struct {
		in   []string
		want args
	}{
		{[]string{"pierre"}, args{cmd: "copy", source: "pierre", dest: "."}},
		{[]string{"pierre", "my-app"}, args{cmd: "copy", source: "pierre", dest: "my-app"}},
		{[]string{"pierre", ".", "--force", "-y"}, args{cmd: "copy", source: "pierre", dest: ".", force: true, yes: true}},
		{[]string{"--name", "x", "pierre"}, args{cmd: "copy", source: "pierre", dest: ".", name: "x"}},
		{[]string{"pierre", "--name=x", "--no-setup"}, args{cmd: "copy", source: "pierre", dest: ".", name: "x", noSetup: true}},
		{[]string{"-n", "x", "-f", "pierre"}, args{cmd: "copy", source: "pierre", dest: ".", name: "x", force: true}},
		{[]string{"--", "-weird"}, args{cmd: "copy", source: "-weird", dest: "."}},
		{[]string{"list"}, args{cmd: "list"}},
		{[]string{"version"}, args{cmd: "version"}},
		{[]string{"--version"}, args{cmd: "version"}},
		{[]string{"-h"}, args{cmd: "help"}},
		{[]string{"help"}, args{cmd: "help"}},
	}
	for _, c := range cases {
		got, err := parseArgs(c.in)
		if err != nil {
			t.Errorf("parseArgs(%v) error: %v", c.in, err)
			continue
		}
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("parseArgs(%v) = %+v, want %+v", c.in, got, c.want)
		}
	}
}

func TestParseArgsErrors(t *testing.T) {
	for _, in := range [][]string{{}, {"a", "b", "c"}, {"--name"}, {"--bogus", "x"}, {"list", "extra"}} {
		if _, err := parseArgs(in); err == nil {
			t.Errorf("parseArgs(%v) expected error", in)
		}
	}
}
