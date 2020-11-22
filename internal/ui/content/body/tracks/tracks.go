package tracks

import (
	"github.com/diamondburned/aqours/internal/durafmt"
	"github.com/diamondburned/aqours/internal/muse/playlist"
	"github.com/gotk3/gotk3/gtk"
	"github.com/gotk3/gotk3/pango"
)

type ParentController interface {
	PlayTrack(p *playlist.Playlist, index int)
	UpdateTracks(p *playlist.Playlist)
}

type ListStorer interface {
	Set(iter *gtk.TreeIter, columns []int, values []interface{}) error
	SetValue(iter *gtk.TreeIter, column int, value interface{}) error
}

type columnType = int

const (
	columnTitle columnType = iota
	columnArtist
	columnAlbum
	columnTime
	columnSelected
)

type Container struct {
	gtk.Stack
	parent ParentController

	Lists map[string]*TrackList // tree model

	// current treeview playlist name
	current string
}

func NewContainer(parent ParentController) *Container {
	stack, _ := gtk.StackNew()
	stack.SetTransitionType(gtk.STACK_TRANSITION_TYPE_CROSSFADE)
	stack.SetTransitionDuration(25)
	stack.Show()

	return &Container{
		parent: parent,
		Stack:  *stack,
		Lists:  map[string]*TrackList{},
	}
}

func (c *Container) SelectPlaylist(playlist *playlist.Playlist) *TrackList {
	pl, ok := c.Lists[playlist.Name]
	if !ok {
		pl = NewTrackList(c.parent, playlist)
		c.Lists[playlist.Name] = pl
		c.Stack.AddNamed(pl, playlist.Name)
	}

	c.current = playlist.Name
	c.Stack.SetVisibleChild(pl)
	return pl
}

func (c *Container) DeletePlaylist(name string) {
	pl, ok := c.Lists[name]
	if !ok {
		return
	}

	c.current = ""
	c.Stack.Remove(pl)
	delete(c.Lists, name)
}

func newColumn(text string, col columnType) *gtk.TreeViewColumn {
	r, _ := gtk.CellRendererTextNew()
	r.SetProperty("weight-set", true)
	r.SetProperty("ellipsize", pango.ELLIPSIZE_END)
	r.SetProperty("ellipsize-set", true)

	c, _ := gtk.TreeViewColumnNewWithAttribute(text, r, "text", int(col))
	c.AddAttribute(r, "weight", int(columnSelected))
	c.SetSizing(gtk.TREE_VIEW_COLUMN_FIXED)
	c.SetResizable(true)

	switch col {
	case columnTime:
		c.SetMinWidth(50)

	case columnSelected:
		c.SetVisible(false)

	default:
		c.SetExpand(true)
		c.SetMinWidth(150)
	}

	return c
}

type TrackRow struct {
	Bold bool
	Iter *gtk.TreeIter
}

func (row *TrackRow) SetBold(store ListStorer, bold bool) {
	row.Bold = bold
	store.SetValue(row.Iter, columnSelected, weight(row.Bold))
}

func (row *TrackRow) setListStore(t *playlist.Track, store ListStorer) {
	store.Set(
		row.Iter,
		[]int{columnTitle, columnArtist, columnAlbum, columnTime, columnSelected},
		[]interface{}{t.Title, t.Artist, t.Album, durafmt.Format(t.Length), weight(row.Bold)},
	)
}

func weight(bold bool) pango.Weight {
	if bold {
		return pango.WEIGHT_BOLD
	}
	return pango.WEIGHT_BOOK
}
