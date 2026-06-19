package util

import (
	"github.com/spf13/pflag"
)

func Flag(l string, s string) pflag.Flag {
	return pflag.Flag{
		Name:      l,
		Hidden:    false,
		Value:     nil,
		Usage:     l,
		Shorthand: s,
		DefValue:  "",
	}
}

func Opt(l string, s string) pflag.Flag {
	return pflag.Flag{
		Name:      l,
		Hidden:    true,
		Value:     nil,
		Usage:     l,
		Shorthand: s,
		DefValue:  "",
	}
}
