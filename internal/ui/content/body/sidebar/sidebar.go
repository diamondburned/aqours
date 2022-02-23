package sidebar

import "github.com/diamondburned/gotk4/pkg/gtk/v4"

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

	scroll := gtk.NewScrolledWindow()
	scroll.SetVExpand(true)
	scroll.SetPolicy(gtk.PolicyAutomatic, gtk.PolicyAutomatic)
	scroll.SetChild(list)

	separator := gtk.NewSeparator(gtk.OrientationVertical)

	aart := NewAlbumArt()

	box := gtk.NewBox(gtk.OrientationVertical, 0)
	box.Append(scroll)
	box.Append(separator)
	box.Append(aart)

	return &Container{
		Box:          *box,
		ListScroll:   scroll,
		PlaylistList: list,
		AlbumArt:     aart,
	}
}
