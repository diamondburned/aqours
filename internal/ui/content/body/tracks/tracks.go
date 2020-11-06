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
	PlayTrack(playlistName string, index int)
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
	gtk.TreeView
	Lists map[string]*TrackList // tree model

	// current treeview playlist name
	current string
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
	tree.AppendColumn(newColumn("", columnSelected))
	tree.Show()
	c.TreeView = *tree

	tree.Connect("row-activated", func(_ *gtk.TextView, path *gtk.TreePath) {
		parent.PlayTrack(c.current, path.GetIndices()[0])
	})

	return c
}

func (c *Container) SelectPlaylist(name string) *TrackList {
	pl, ok := c.Lists[name]
	if !ok {
		pl = NewTrackList()
		c.Lists[name] = pl
	}

	c.current = name
	c.TreeView.SetModel(pl)
	return pl
}

func (c *Container) DeletePlaylist(name string) {
	if _, ok := c.Lists[name]; !ok {
		return
	}

	c.current = ""
	c.TreeView.SetModel(nil)
	delete(c.Lists, name)
}

func newColumn(text string, col columnType) *gtk.TreeViewColumn {
	r, _ := gtk.CellRendererTextNew()
	// r.SetProperty("weight", pango.WEIGHT_ULTRABOLD)
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

type TrackPath = string

type TrackList struct {
	gtk.ListStore
	Paths  []string
	Tracks map[TrackPath]*Track

	playing *Track
}

func NewTrackList() *TrackList {
	store, _ := gtk.ListStoreNew(
		glib.TYPE_STRING, // columnTitle
		glib.TYPE_STRING, // columnArtist
		glib.TYPE_STRING, // columnAlbum
		glib.TYPE_STRING, // columnTime
		glib.TYPE_INT,    // columnSelected - pango.Weight
	)

	return &TrackList{
		ListStore: *store,
		Tracks:    map[TrackPath]*Track{},
	}
}

func (list *TrackList) SetPlaying(playing *Track) {
	if list.playing != nil {
		list.playing.SetBold(list, false)
	}

	list.playing = playing
	list.playing.SetBold(list, true)
}

func (list *TrackList) SetTracks(tracks []*playlist.Track) {
	for _, track := range tracks {
		// Skip existing tracks.
		if _, ok := list.Tracks[track.Filepath]; ok {
			continue
		}

		advTrack := &Track{
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
				wrapTrack.Track = updatedTrack
				// Update the list entry as well.
				wrapTrack.setListStore(list)
			}
		})
	})
}

type Track struct {
	*playlist.Track
	Bold bool
	Iter *gtk.TreeIter
}

func (t *Track) SetBold(store ListStorer, bold bool) {
	t.Bold = bold
	store.SetValue(t.Iter, columnSelected, weight(t.Bold))
}

func (t *Track) setListStore(store ListStorer) {
	store.Set(
		t.Iter,
		[]int{columnTitle, columnArtist, columnAlbum, columnTime, columnSelected},
		[]interface{}{t.Title, t.Artist, t.Album, durafmt.Format(t.Length), weight(t.Bold)},
	)
}

func weight(bold bool) pango.Weight {
	if bold {
		return pango.WEIGHT_BOLD
	}
	return pango.WEIGHT_BOOK
}
