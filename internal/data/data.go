package data

import (
	"encoding/json"
	"os"

	"go.lsp.dev/protocol"
)

type Data map[string]interface{}

func Workspaces() []protocol.WorkspaceFolder {
	var file string
	var data Data
	var workspaces []protocol.WorkspaceFolder = []protocol.WorkspaceFolder{}
	locations := []string{
		"$HOME/.local/share/nvim/down.json",
		"$HOME/.config/down/down.json",
		"$HOME/.down/down.json",
	}
	for _, l := range locations {
		if _, err := os.Stat(l); err == nil {
			file = l
			break
		}
	}
	fc, err := os.ReadFile(file)
	if err != nil {
		return workspaces
	}
	if json.Unmarshal(fc, &data) != nil {
		return workspaces
	}
	for k, v := range data {
		if k == "workspace.workspace_folders" {
			for _, w := range v.([]interface{}) {
				workspaces = append(workspaces, w.(protocol.WorkspaceFolder))
			}
		}
	}
	return workspaces
}
