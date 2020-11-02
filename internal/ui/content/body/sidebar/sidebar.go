package sidebar

import "github.com/gotk3/gotk3/gtk"

type ParentController interface {
	SelectPlaylist(name string)
}

type Container struct {
	gtk.Box

	ListScroll   *gtk.ScrolledWindow
	PlaylistList *PlaylistList

	AlbumArt *AlbumArt
}

func NewContainer(parent ParentController) *Container {
	list := NewPlaylistList(parent)
	list.Show()

	scroll, _ := gtk.ScrolledWindowNew(nil, nil)
	scroll.SetPolicy(gtk.POLICY_AUTOMATIC, gtk.POLICY_AUTOMATIC)
	scroll.Add(list)
	scroll.Show()

	separator, _ := gtk.SeparatorNew(gtk.ORIENTATION_VERTICAL)
	separator.Show()

	aart := NewAlbumArt()

	box, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	box.PackStart(scroll, true, true, 0)
	box.PackStart(separator, false, false, 0)
	box.PackStart(aart, false, false, 0)
	box.Show()

	return &Container{
		Box:          *box,
		ListScroll:   scroll,
		PlaylistList: list,
		AlbumArt:     aart,
	}
}
