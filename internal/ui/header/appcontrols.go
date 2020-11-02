package header

import (
	"os"

	"github.com/diamondburned/aqours/internal/muse/playlist"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
)

type AppControls struct {
	gtk.Box
	OpenPlaylistButton gtk.Button
}

func NewAppControls(parent ParentController) *AppControls {
	openBtn, _ := gtk.ButtonNewFromIconName("document-open-symbolic", gtk.ICON_SIZE_BUTTON)
	openBtn.Connect("clicked", func() { spawnChooser(parent) })
	openBtn.SetTooltipMarkup("Add Playlist")
	openBtn.Show()

	box, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 5)
	box.PackStart(openBtn, false, false, 0)
	box.Show()

	return &AppControls{
		Box:                *box,
		OpenPlaylistButton: *openBtn,
	}
}

func spawnChooser(parent ParentController) {
	dialog, _ := gtk.FileChooserDialogNewWith2Buttons(
		"Choose Playlist", parent.ToWindow(),
		gtk.FILE_CHOOSER_ACTION_OPEN,
		"Cancel", gtk.RESPONSE_CANCEL,
		"Add", gtk.RESPONSE_ACCEPT,
	)

	p, err := os.Getwd()
	if err != nil {
		p = glib.GetUserDataDir()
	}

	ff, _ := gtk.FileFilterNew()
	ff.SetName("Playlists")
	for _, ext := range playlist.SupportedExtensions() {
		ff.AddPattern("*" + ext)
	}

	dialog.SetFilter(ff)
	dialog.SetLocalOnly(false)
	dialog.SetCurrentFolder(p)
	dialog.SetSelectMultiple(false)

	defer dialog.Close()

	if res := dialog.Run(); res != gtk.RESPONSE_ACCEPT {
		return
	}

	names, _ := dialog.GetFilenames()
	if len(names) == 1 {
		parent.AddPlaylist(names[0])
	}
}
