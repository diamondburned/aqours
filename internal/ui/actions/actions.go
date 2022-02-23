package actions

import (
	"log"
	"strings"

	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
)

// Stateful is a stateful action group, which would allow additional methods
// that would otherwise be impossible to do with a simple Action Map.
type Stateful struct {
	*gio.SimpleActionGroup
	labels []string // labels
}

func NewStateful() *Stateful {
	group := gio.NewSimpleActionGroup()
	return &Stateful{
		SimpleActionGroup: group,
	}
}

func (s *Stateful) Reset() {
	for _, label := range s.labels {
		s.RemoveAction(ActionName(label))
	}
	s.labels = nil
}

// Len returns the number of menu entries.
func (s *Stateful) Len() int {
	return len(s.labels)
}

func (s *Stateful) AddAction(label string, call func()) {
	sa := gio.NewSimpleAction(ActionName(label), nil)
	sa.ConnectActivate(func(*glib.Variant) { call() })

	s.labels = append(s.labels, label)
	s.SimpleActionGroup.AddAction(sa)
}

func (s *Stateful) LookupAction(label string) gio.Actioner {
	for _, l := range s.labels {
		if l == label {
			return s.SimpleActionGroup.LookupAction(ActionName(label))
		}
	}
	return nil
}

func (s *Stateful) RemoveAction(label string) {
	for i, l := range s.labels {
		if l == label {
			s.labels = append(s.labels[:i], s.labels[:i+1]...)
			s.SimpleActionGroup.RemoveAction(ActionName(label))
			return
		}
	}
}

// ActionName converts the label name into the action name.
func ActionName(label string) (actionName string) {
	actionName = strings.Replace(label, " ", "-", -1)

	if !gio.ActionNameIsValid(actionName) {
		log.Panicf("Label makes for invalid action name %q\n", actionName)
	}

	return
}
