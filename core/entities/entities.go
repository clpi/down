package entities

import (
	// "date"
	// "fmt"

	// protocol "github.com/tliron/glsp/protocol_3_16"
	// "golang.org/x/text/date"
	// "google.golang.org/genproto/googleapis/type/datetime"
	"time"
)

type (
	Flag struct {
		Id        string `url:"id,omitempty" json:"id,omitempty"`
		Flag      string `url:"flag,omitempty" json:"flag,omitempty"`
		Callback  string `url:"callback,omitempty" json:"callback,omitempty"`
		Condition string `url:"condition,omitempty" json:"condition,omitempty"`
		Children  []Flag `url:"children,omitempty" json:"children,omitempty"`
	}
	Tag struct {
		Tag     string    `url:"tag,omitempty" json:"tag,omitempty"`
		Flags   []Flag    `url:"flags,omitempty" json:"flags,omitempty"`
		Created time.Time `url:"created,omitempty" json:"created,omitempty"`
	}
	Entry struct {
		Id   string `url:"id,omitempty" json:"id,omitempty"`
		Name string `url:"name,omitempty" json:"name,omitempty"`
	}
	Log struct {
		Id      string `url:"id,omitempty" json:"id,omitempty"`
		Name    string `url:"name,omitempty" json:"name,omitempty"`
		Desc    string `url:"desc,omitempty" json:"desc,omitempty"`
		Created string `url:"created,omitempty" json:"created,omitempty"`
		Entries []Entry
	}
	Project struct {
		Id   string `url:"webhook,omitempty" json:"webhook,omitempty"`
		Tags []Tag
	}
)
