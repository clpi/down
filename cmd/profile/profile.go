package profile

import (
	"fmt"
	"log"

	coreprofile "github.com/clpi/down/core/profile"
	"github.com/spf13/cobra"
)

var Profile = cobra.Command{
	Use:     "profile",
	Aliases: []string{"prof", "user"},
	Short:   "Manage user profile and preferences",
	Long:    "View and manage user profile, preferences, and AI settings",
	Run: func(cmd *cobra.Command, args []string) {
		p, err := coreprofile.LoadProfile()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(p.Summary())
	},
}

var profileSet = cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a profile preference",
	Args:  cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		p, err := coreprofile.LoadProfile()
		if err != nil {
			log.Fatal(err)
		}
		key := args[0]
		value := args[1]

		// Handle boolean values
		var val interface{}
		switch value {
		case "true":
			val = true
		case "false":
			val = false
		default:
			val = value
		}

		if err := p.SetPreference(key, val); err != nil {
			log.Fatal(err)
		}
		if err := p.Save(); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Set %s = %v\n", key, val)
	},
}

var profileInit = cobra.Command{
	Use:   "init <name>",
	Short: "Initialize a new user profile",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		p := coreprofile.NewProfile(name)
		if err := p.Save(); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Profile created: %s\n", name)
	},
}

func init() {
	Profile.AddCommand(&profileSet)
	Profile.AddCommand(&profileInit)
}
