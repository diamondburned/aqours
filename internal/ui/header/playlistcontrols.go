package header

import (
	"github.com/diamondburned/aqours/internal/gtkutil"
	"github.com/diamondburned/aqours/internal/ui/actions"
	"github.com/diamondburned/aqours/internal/ui/css"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

var renameEntryCSS = css.PrepareClass("rename-entry", `
	entry.rename-entry {
		margin: 8px;
	}
`)

type ParentPlaylistController interface {
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
	// SortSelectedTracks sorts the selected songs.
	SortSelectedTracks()
}

type PlaylistControls struct {
	gtk.Revealer
	Hamburger *actions.MenuButton
	HamMenu   *actions.Menu
}

func NewPlaylistControls(parent ParentPlaylistController) *PlaylistControls {
	hamMenu := actions.NewMenu("playlist-ctrl")

	hamburger := actions.NewMenuButton()
	hamburger.SetIconName("open-menu-symbolic")
	hamburger.Bind(hamMenu)

	rev := gtk.NewRevealer()
	rev.SetRevealChild(false)
	rev.SetTransitionDuration(50)
	rev.SetTransitionType(gtk.RevealerTransitionTypeCrossfade)
	rev.SetChild(hamburger)

	hamMenu.AddAction("Rename Playlist", func() { spawnRenameDialog(parent) })
	hamMenu.AddAction("Save Playlist", parent.SaveCurrentPlaylist)
	hamMenu.AddAction("Sort Selected Tracks", parent.SortSelectedTracks)

	return &PlaylistControls{
		Revealer:  *rev,
		Hamburger: hamburger,
		HamMenu:   hamMenu,
	}
}

const nameCollideMsg = "Playlist already exists with the same name."

func spawnRenameDialog(parent ParentPlaylistController) {
	window := gtkutil.ActiveWindow()
	dialog := gtk.NewDialogWithFlags(
		"Rename Playlist", window, gtk.DialogModal|gtk.DialogUseHeaderBar)

	dialog.AddButton("Rename", int(gtk.ResponseApply))
	// We're starting w/ the same name, so we shouldn't let it apply.
	dialog.SetResponseSensitive(int(gtk.ResponseApply), false)

	var newName string

	entry := gtk.NewEntry()
	entry.SetText(parent.PlaylistName())
	entry.SetPlaceholderText("New Playlist")
	entry.Connect("changed", func() {
		t := entry.Text()
		if t == "" || parent.HasPlaylist(t) {
			dialog.SetResponseSensitive(int(gtk.ResponseApply), false)
			entry.SetIconFromIconName(gtk.EntryIconSecondary, "dialog-error-symbolic")
			entry.SetIconTooltipText(gtk.EntryIconSecondary, nameCollideMsg)
			newName = ""
		} else {
			dialog.SetResponseSensitive(int(gtk.ResponseApply), true)
			entry.SetIconFromIconName(gtk.EntryIconSecondary, "")
			newName = t
		}
	})

	renameEntryCSS(entry)

	c := dialog.ContentArea()
	c.Append(entry)

	dialog.ConnectResponse(func(res int) {
		defer dialog.Destroy()

		if res != int(gtk.ResponseApply) || newName == "" {
			return
		}

		parent.RenamePlaylist(newName)
	})
}
