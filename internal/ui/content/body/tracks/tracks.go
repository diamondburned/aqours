package tracks

import (
	"log"

	"github.com/diamondburned/aqours/internal/durafmt"
	"github.com/diamondburned/aqours/internal/muse/playlist"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
	"github.com/gotk3/gotk3/pango"
)

type ParentController interface {
	PlayTrack(*TrackList, int)
}

type ListStorer interface {
	Set(iter *gtk.TreeIter, columns []int, values []interface{}) error
}

type columnType = int

const (
	columnTitle columnType = iota
	columnArtist
	columnAlbum
	columnTime
)

type Container struct {
	gtk.TreeView
	Lists   map[string]*TrackList // tree model
	Current *TrackList
}

func NewContainer(parent ParentController) *Container {
	c := &Container{
		Lists: map[string]*TrackList{},
	}

	tree, _ := gtk.TreeViewNew()
	tree.SetActivateOnSingleClick(false)
	tree.AppendColumn(newColumn("Title", columnTitle))
	tree.AppendColumn(newColumn("Artist", columnArtist))
	tree.AppendColumn(newColumn("Album", columnAlbum))
	tree.AppendColumn(newColumn("", columnTime))
	tree.Show()
	c.TreeView = *tree

	tree.Connect("row-activated", func(tv *gtk.TextView, path *gtk.TreePath) {
		parent.PlayTrack(c.Current, path.GetIndices()[0])
	})

	return c
}

func (c *Container) SelectPlaylist(name string) *TrackList {
	pl, ok := c.Lists[name]
	if !ok {
		pl = NewTrackList()
		c.Lists[name] = pl
	}

	c.Current = pl
	c.TreeView.SetModel(pl)
	return pl
}

func (c *Container) DeletePlaylist(name string) {
	if _, ok := c.Lists[name]; !ok {
		return
	}

	c.Current = nil
	c.TreeView.SetModel(nil)
	delete(c.Lists, name)
}

func newColumn(text string, col columnType) *gtk.TreeViewColumn {
	r, _ := gtk.CellRendererTextNew()
	r.SetProperty("ellipsize", pango.ELLIPSIZE_END)
	r.SetProperty("ellipsize-set", true)

	c, _ := gtk.TreeViewColumnNewWithAttribute(text, r, "text", int(col))
	c.SetSizing(gtk.TREE_VIEW_COLUMN_FIXED)
	c.SetResizable(true)

	if col != columnTime {
		c.SetExpand(true)
		c.SetMinWidth(150)
	} else {
		c.SetMinWidth(50)
	}

	return c
}

type TrackPath = string

type TrackList struct {
	gtk.ListStore
	Paths  []string
	Tracks map[TrackPath]Track
}

func NewTrackList() *TrackList {
	store, _ := gtk.ListStoreNew(
		glib.TYPE_STRING, // columnTitle
		glib.TYPE_STRING, // columnArtist
		glib.TYPE_STRING, // columnAlbum
		glib.TYPE_STRING, // columnTime
	)

	return &TrackList{
		ListStore: *store,
		Tracks:    map[TrackPath]Track{},
	}
}

func (list *TrackList) SetTracks(tracks []*playlist.Track) {
	for _, track := range tracks {
		// Skip existing tracks.
		if _, ok := list.Tracks[track.Filepath]; ok {
			continue
		}

		advTrack := Track{
			Track: track,
			Iter:  list.ListStore.Append(),
		}

		advTrack.setListStore(list)
		list.Paths = append(list.Paths, track.Filepath)
		list.Tracks[track.Filepath] = advTrack
	}

	// TODO: this has a cache stampede problem. We need to have a context to
	// cancel this.
	go playlist.BatchProbe(tracks, func(updatedTrack *playlist.Track, err error) {
		if err != nil {
			log.Println("Failed to probe:", err)
			return
		}

		glib.IdleAdd(func() {
			wrapTrack, ok := list.Tracks[updatedTrack.Filepath]
			if ok {
				// Update the underneath struct value, not the pointer itself.
				*wrapTrack.Track = *updatedTrack
				// Update the list entry as well.
				wrapTrack.setListStore(list)
			}
		})
	})
}

type Track struct {
	*playlist.Track
	Iter *gtk.TreeIter
}

func (t Track) setListStore(store ListStorer) {
	store.Set(
		t.Iter,
		[]int{columnTitle, columnArtist, columnAlbum, columnTime},
		[]interface{}{t.Title, t.Artist, t.Album, durafmt.Format(t.Length)},
	)
}
