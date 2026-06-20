package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/clpi/down/cmd/wsutil"
	"github.com/clpi/down/lsp"
	"github.com/spf13/cobra"
)

var configRoot string

var Config = cobra.Command{
	Use:     "config",
	Aliases: []string{"cfg", "conf", "c"},
	Short:   "View or edit workspace configuration",
	Version: lsp.Version,
	Run: func(cmd *cobra.Command, args []string) {
		root := wsutil.ResolveRoot(configRoot)
		cfgPath := filepath.Join(root, ".down", "down.json")
		data, err := os.ReadFile(cfgPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "No config at %s (run `down init` first)\n", cfgPath)
			os.Exit(1)
		}
		fmt.Println(string(data))
	},
}

var configGet = cobra.Command{
	Use:   "get <key>",
	Short: "Get a config value from down.json",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		m := loadConfigMap()
		key := args[0]
		if v, ok := m[key]; ok {
			fmt.Println(v)
		} else {
			fmt.Fprintf(os.Stderr, "key not found: %s\n", key)
			os.Exit(1)
		}
	},
}

var configSet = cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a config value in down.json",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		m := loadConfigMap()
		m[args[0]] = args[1]
		saveConfigMap(m)
		fmt.Printf("Set %s = %s\n", args[0], args[1])
	},
}

func loadConfigMap() map[string]interface{} {
	root := wsutil.ResolveRoot(configRoot)
	cfgPath := filepath.Join(root, ".down", "down.json")
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "No config at %s\n", cfgPath)
		os.Exit(1)
	}
	var m map[string]interface{}
	if json.Unmarshal(data, &m) != nil {
		m = map[string]interface{}{}
	}
	return m
}

func saveConfigMap(m map[string]interface{}) {
	root := wsutil.ResolveRoot(configRoot)
	cfgPath := filepath.Join(root, ".down", "down.json")
	data, _ := json.MarshalIndent(m, "", "  ")
	os.MkdirAll(filepath.Dir(cfgPath), 0755)
	os.WriteFile(cfgPath, data, 0644)
}

func init() {
	Config.PersistentFlags().StringVar(&configRoot, "root", "", "Workspace root")
	Config.AddCommand(&configGet)
	Config.AddCommand(&configSet)
}
