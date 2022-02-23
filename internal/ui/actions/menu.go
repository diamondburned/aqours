package actions

import (
	"fmt"

	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

type Menu struct {
	*Stateful
	menu   *gio.Menu
	prefix string
}

func NewMenu(prefix string) *Menu {
	return &Menu{
		Stateful: NewStateful(), // actiongroup and menu not linked
		menu:     gio.NewMenu(),
		prefix:   prefix,
	}
}

func (m *Menu) Prefix() string {
	return m.prefix
}

func (m *Menu) MenuModel() (string, *gio.MenuModel) {
	return m.prefix, &m.menu.MenuModel
}

func (m *Menu) InsertActionGroup(widget gtk.Widgetter) {
	w := gtk.BaseWidget(widget)
	w.InsertActionGroup(m.prefix, m)
}

func (m *Menu) ButtonRightClick(w gtk.Widgetter) {
	click := gtk.NewGestureClick()
	click.SetButton(gdk.BUTTON_SECONDARY)
	click.ConnectPressed(func(n int, x, y float64) {
		m.Popup(w)
	})
}

// Popup pops up the menu popover. It does not pop up anything if there are no
// menu items.
func (m *Menu) Popup(relative gtk.Widgetter) {
	p := m.popover(relative)
	if p == nil || m.Len() == 0 {
		return
	}

	p.Popup()
}

func (m *Menu) popover(relative gtk.Widgetter) *gtk.PopoverMenu {
	_, model := m.MenuModel()

	p := gtk.NewPopoverMenuFromModel(model)
	p.SetParent(relative)
	p.SetPosition(gtk.PosRight)

	return p
}

func (m *Menu) Reset() {
	m.menu.RemoveAll()
	m.Stateful.Reset()
}

func (m *Menu) AddAction(label string, call func()) {
	m.Stateful.AddAction(label, call)
	m.menu.Append(label, fmt.Sprintf("%s.%s", m.prefix, ActionName(label)))
}

func (m *Menu) RemoveAction(label string) {
	var labels = m.Stateful.labels

	for i, l := range labels {
		if l == label {
			labels = append(labels[:i], labels[:i+1]...)
			m.menu.Remove(i)

			m.Stateful.labels = labels
			m.Stateful.SimpleActionGroup.RemoveAction(ActionName(label))

			return
		}
	}
}
