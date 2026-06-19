package entities

import (
	"time"

	"github.com/clpi/down/lsp/util"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

type (
	Agenda struct {
		Id       string
		Title    string
		Info     Info
		Due      time.Time
		Priority int
		Status   int
		Sections []Agenda
		Tasks    []Task
		// Patterns to check for in tasks to automatically add to agenda
		Patterns []string
		Globs    []string
	}
	Info struct {
		About    string
		Header   string
		Parent   any
		Tags     []Tag
		Created  time.Time
		File     File
		Project  []string
		Position protocol.Position
	}
	Task struct {
		Id       string `url:"id,omitempty" json:"id,omitempty"`
		Name     string
		Info     Info
		Priority int
		Repeat   Repeat
		Due      time.Time
		Status   int
		Subtasks []Task
	}
)

func (t *Task) setPriority(p int) {
	t.Priority = p
}
func (t *Task) tag(tag string) {
	t.Info.Tags = append(t.Info.Tags, Tag{
		Tag:     tag,
		Created: time.Now(),
		Flags:   []Flag{},
	})
}
func (t *Task) addTag(tag Tag) {
	t.Info.Tags = append(t.Info.Tags, tag)
}
func (t *Task) Cancel() {
	t.Status = -1
}

func (t *Task) Complete() {
	t.Status = 1
}
func (t *Task) Reschedule(d Date) {
	t.Due = time.Now()
}
func (t *Task) Subinfo() Info {
	return Info{
		Created:  time.Now(),
		File:     t.Info.File,
		Position: util.NextLine(t.Info.Position.Line, t.Info.Position.Character),
		Header:   t.Info.Header,
		Parent:   t.Info,
		Project:  t.Info.Project,
	}
}
func (t *Task) defaultSubtask(task string) Task {
	return Task{
		Id:       task,
		Info:     t.Subinfo(),
		Name:     task,
		Due:      t.Due,
		Status:   0,
		Priority: t.Priority,
		Subtasks: []Task{},
	}
}
func (t *Task) addSubtask(task string) {
	t.addSubtaskWith(t.defaultSubtask(task))
}
func (t *Task) addSubtaskWith(sub Task) {
	t.Subtasks = append(t.Subtasks, sub)
}
