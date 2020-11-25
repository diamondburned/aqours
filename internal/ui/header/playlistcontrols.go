package header

import (
	"log"

	"github.com/diamondburned/aqours/internal/ui/actions"
	"github.com/diamondburned/aqours/internal/ui/css"
	"github.com/gotk3/gotk3/gtk"
)

var renameEntryCSS = css.PrepareClass("rename-entry", `
	entry.rename-entry {
		margin: 8px;
	}
`)

type ParentPlaylistController interface {
	gtk.IWindow
	// RenamePlaylist renames the current playlist.
	RenamePlaylist(newName string)
	// HasPlaylist returns true if the playlist already exists with the given
	// name.
	HasPlaylist(name string) bool
	// PlaylistName gets the playlist name.
	PlaylistName() string
	// GoBack navigates the body leaflet to the left panel.
	GoBack()
	// SaveCurrentPlaylist saves the current playlist and marks the playlist
	// name as saved.
	SaveCurrentPlaylist()
}

type PlaylistControls struct {
	gtk.Revealer
	Hamburger *actions.MenuButton
	HamMenu   *actions.Menu
}

func NewPlaylistControls(parent ParentPlaylistController) *PlaylistControls {
	hamMenu := actions.NewMenu("playlist-ctrl")

	icon, _ := gtk.ImageNewFromIconName("open-menu-symbolic", gtk.ICON_SIZE_BUTTON)
	icon.Show()

	hamburger := actions.NewMenuButton()
	hamburger.SetImage(icon)
	hamburger.Bind(hamMenu)
	hamburger.Show()

	rev, _ := gtk.RevealerNew()
	rev.SetRevealChild(false)
	rev.SetTransitionDuration(50)
	rev.SetTransitionType(gtk.REVEALER_TRANSITION_TYPE_CROSSFADE)
	rev.Add(hamburger)
	rev.Show()

	hamMenu.AddAction("Rename Playlist", func() { spawnRenameDialog(parent) })
	hamMenu.AddAction("Save Playlist", parent.SaveCurrentPlaylist)

	return &PlaylistControls{
		Revealer:  *rev,
		Hamburger: hamburger,
		HamMenu:   hamMenu,
	}
}

const nameCollideMsg = "Playlist already exists with the same name."

func spawnRenameDialog(parent ParentPlaylistController) {
	dialog, _ := gtk.DialogNewWithButtons(
		"Rename Playlist", parent, gtk.DIALOG_MODAL|gtk.DIALOG_USE_HEADER_BAR,
		[]interface{}{"Rename", gtk.RESPONSE_APPLY},
	)

	// We're starting w/ the same name, so we shouldn't let it apply.
	dialog.SetResponseSensitive(gtk.RESPONSE_APPLY, false)

	var newName string

	entry, _ := gtk.EntryNew()
	entry.SetText(parent.PlaylistName())
	entry.SetPlaceholderText("New Playlist")
	entry.Connect("changed", func() {
		t, err := entry.GetText()
		if err != nil {
			log.Println("Failed to get entry text:", err)
			return
		}

		if t == "" || parent.HasPlaylist(t) {
			dialog.SetResponseSensitive(gtk.RESPONSE_APPLY, false)
			entry.SetIconFromIconName(gtk.ENTRY_ICON_SECONDARY, "dialog-error-symbolic")
			entry.SetIconTooltipText(gtk.ENTRY_ICON_SECONDARY, nameCollideMsg)
		} else {
			dialog.SetResponseSensitive(gtk.RESPONSE_APPLY, true)
			entry.RemoveIcon(gtk.ENTRY_ICON_SECONDARY)
			newName = t
		}
	})
	entry.Show()
	renameEntryCSS(entry)

	c, _ := dialog.GetContentArea()
	c.Add(entry)
	c.Show()

	defer dialog.Close()

	if res := dialog.Run(); res != gtk.RESPONSE_APPLY || newName == "" {
		return
	}

	parent.RenamePlaylist(newName)
}
