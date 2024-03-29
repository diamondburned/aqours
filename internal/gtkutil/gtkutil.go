package gtkutil

import (
	"log"
	"strings"

	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

// TODO: move these out of internal/ui/actions
// TODO: move internal/ui/actions out of internal/ui

// BindActionMap binds the given map of actions (of key prefixed appropriately)
// to the given widget.
func BindActionMap(wd gtk.Widgetter, m map[string]func()) {
	actions := make(map[string]*gio.SimpleActionGroup)
	w := gtk.BaseWidget(wd)

	for k, v := range m {
		parts := strings.SplitN(k, ".", 2)
		if len(parts) != 2 {
			log.Panicf("invalid action key %q", k)
		}

		group, ok := actions[parts[0]]
		if !ok {
			group = gio.NewSimpleActionGroup()
			w.InsertActionGroup(parts[0], group)
			actions[parts[0]] = group
		}

		group.AddAction(ActionFunc(parts[1], v))
	}
}

// ActionFunc creates a CallbackActionFunc from a function.
func ActionFunc(name string, f func()) *gio.SimpleAction {
	c := gio.NewSimpleAction(name, nil)
	c.ConnectActivate(func(*glib.Variant) { f() })
	return c
}

// MenuPair creates a gtk.Menu out of the given menu pair. The returned Menu
// instance satisfies gio.MenuModeller. The first value of a pair should be the
// name.
func MenuPair(pairs [][2]string) *gio.Menu {
	menu := gio.NewMenu()
	for _, pair := range pairs {
		menu.Append(pair[0], pair[1])
	}
	return menu
}

// PopoverWidth is the default popover width.
const PopoverWidth = 150

// NewPopoverMenu creates a new Popover menu.
func NewPopoverMenu(w gtk.Widgetter, pos gtk.PositionType, menu gio.MenuModeller) *gtk.PopoverMenu {
	popover := gtk.NewPopoverMenuFromModel(menu)
	popover.SetParent(w)
	popover.SetPosition(pos)
	popover.SetSizeRequest(PopoverWidth, -1)
	popover.SetMnemonicsVisible(true)
	popover.ConnectClosed(func() {
		glib.TimeoutSecondsAdd(5, popover.Unparent)
	})
	return popover
}

// NewPopoverMenuAt is a convenient function for NewPopoverMenu and
// SetPointingTo.
func NewPopoverMenuAt(w gtk.Widgetter, pos gtk.PositionType, x, y float64, menu gio.MenuModeller) *gtk.PopoverMenu {
	rect := gdk.NewRectangle(int(x), int(y), 0, 0)
	p := NewPopoverMenu(w, pos, menu)
	p.SetPointingTo(&rect)
	return p
}

// BindPopoverMenu binds the menu popover at the given position for the given
// widget.
func BindPopoverMenu(wd gtk.Widgetter, pos gtk.PositionType, menu gio.MenuModeller) {
	BindRightClick(wd, func(x, y float64) {
		p := NewPopoverMenuAt(wd, pos, x, y, menu)
		p.Popup()
	})
}

func BindRightClick(wd gtk.Widgetter, f func(x, y float64)) {
	rclick := gtk.NewGestureClick()
	rclick.SetExclusive(true)
	rclick.SetButton(gdk.BUTTON_SECONDARY)
	rclick.ConnectPressed(func(n int, x, y float64) { f(x, y) })

	w := gtk.BaseWidget(wd)
	w.AddController(rclick)
}

// ActiveWindow returns the active window.
func ActiveWindow() *gtk.Window {
	app := gio.ApplicationGetDefault().Cast().(*gtk.Application)
	if app != nil {
		return app.ActiveWindow()
	}

	windowList := gtk.WindowGetToplevels()

	for i := uint(0); true; i++ {
		window := windowList.Item(i)
		if window == nil {
			break
		}

		win := window.Cast().(*gtk.Window)
		if !win.IsActive() {
			continue
		}

		return win
	}

	return nil
}
