package header

import (
	"os"

	"github.com/diamondburned/aqours/internal/gtkutil"
	"github.com/diamondburned/aqours/internal/muse/playlist"
	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

type AppControls struct {
	*gtk.Box
	OpenPlaylistButton *gtk.Button
}

func NewAppControls(parent ParentController) *AppControls {
	openBtn := gtk.NewButtonFromIconName("list-add-symbolic")
	openBtn.ConnectClicked(func() { spawnChooser(parent) })
	openBtn.SetTooltipMarkup("Add Playlist")

	box := gtk.NewBox(gtk.OrientationHorizontal, 5)
	box.Append(openBtn)

	return &AppControls{
		Box:                box,
		OpenPlaylistButton: openBtn,
	}
}

func spawnChooser(parent ParentController) {
	dialog := gtk.NewFileChooserNative(
		"Choose Playlist", gtkutil.ActiveWindow(),
		gtk.FileChooserActionOpen, "Add", "Cancel",
	)

	p, err := os.Getwd()
	if err != nil {
		p = glib.GetUserDataDir()
	}

	ff := gtk.NewFileFilter()
	ff.SetName("Playlists")
	for _, ext := range playlist.SupportedExtensions() {
		ff.AddPattern("*" + ext)
	}

	dialog.SetFilter(ff)
	dialog.SetCurrentFolder(gio.NewFileForPath(p))
	dialog.SetSelectMultiple(false)
	dialog.ConnectResponse(func(id int) {
		defer dialog.Destroy()

		if id != int(gtk.ResponseAccept) {
			return
		}

		fileList := dialog.Files()

		for i := uint(0); true; i++ {
			filer := fileList.Item(i).Cast().(gio.Filer)
			if filer == nil {
				continue
			}

			path := filer.Path()
			if path == "" {
				continue
			}

			parent.AddPlaylist(path)
		}
	})
}
