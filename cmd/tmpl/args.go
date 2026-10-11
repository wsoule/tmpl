package main

import (
	"errors"
	"fmt"
	"strings"
)

type args struct {
	cmd     string // "copy", "list", "version", "help"
	source  string
	dest    string
	name    string
	force   bool
	noSetup bool
	yes     bool
}

// parseArgs accepts flags before or after positional arguments
// (`tmpl pierre . --force`), which the standard flag package does not.
func parseArgs(argv []string) (args, error) {
	var a args
	var pos []string
	for i := 0; i < len(argv); i++ {
		arg := argv[i]
		if arg == "--" {
			pos = append(pos, argv[i+1:]...)
			break
		}
		if !strings.HasPrefix(arg, "-") || arg == "-" {
			pos = append(pos, arg)
			continue
		}
		name, value, hasValue := strings.Cut(arg, "=")
		switch name {
		case "-n", "--name":
			if !hasValue {
				if i+1 >= len(argv) {
					return args{}, errors.New("--name needs a value")
				}
				i++
				value = argv[i]
			}
			a.name = value
		case "-f", "--force":
			a.force = true
		case "--no-setup":
			a.noSetup = true
		case "-y", "--yes":
			a.yes = true
		case "-h", "--help":
			return args{cmd: "help"}, nil
		case "-v", "--version":
			return args{cmd: "version"}, nil
		default:
			return args{}, fmt.Errorf("unknown flag %s", arg)
		}
	}

	if len(pos) == 1 {
		switch pos[0] {
		case "list", "version", "help":
			return args{cmd: pos[0]}, nil
		}
	}
	if len(pos) > 0 && pos[0] == "list" {
		return args{}, errors.New("list takes no arguments")
	}
	switch len(pos) {
	case 0:
		return args{}, errors.New("missing template source")
	case 1:
		a.source, a.dest = pos[0], "."
	case 2:
		a.source, a.dest = pos[0], pos[1]
	default:
		return args{}, errors.New("too many arguments")
	}
	a.cmd = "copy"
	return a, nil
}
