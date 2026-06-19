package util

import "github.com/spf13/cobra"

type (
	SubAnnotations map[string]string
)

func Annotations(a *cobra.Command, b map[string]string) map[string]string {
	ann := make(map[string]string)
	if a.Annotations == nil {
		if a.Parent().Annotations != nil {
			ann = a.Parent().Annotations
		} else if a.Root().Annotations != nil {
			ann = a.Root().Annotations
		}
	} else {
		ann = a.Annotations
	}
	for k, v := range b {
		ann[k] = v
	}
	return ann
}
