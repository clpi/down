package util

import (
	"github.com/spf13/cobra"
)

func Group(id string, l string) cobra.Group {
	return cobra.Group{Title: l, ID: id}
}
