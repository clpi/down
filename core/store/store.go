package store

import (
	"go.lsp.dev/protocol"
)

type (
	StoreIndex interface {
		int64 | string
	}
	Store[K comparable, V any] struct {
		Id        string                              `json:"id"`
		Uri       protocol.URI                        `json:"uri"`
		Kind      string                              `json:"kind"`
		Default   K                                   `json:"default"`
		Name      string                              `json:"name"`
		About     string                              `json:"about"`
		Workspace string                              `json:"workspace"`
		Scope     string                              `json:"scope"`
		Config    LocalConfig[map[string]interface{}] `json:"config"`
		Data      map[K]V                             `json:"data"`
	}
	ListStore[V any] Store[string, V]
)

var (
	Scope = map[string]int64{
		"local":     0,
		"global":    1,
		"workspace": 2,
		"project":   3,
		"user":      4,
		"system":    5,
		"default":   6,
	}
)
