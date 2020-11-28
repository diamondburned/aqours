package tracks

import (
	"log"
	"net/url"
	"strings"

	"github.com/diamondburned/aqours/internal/state"
	"github.com/diamondburned/aqours/internal/state/prober"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
)

type TrackPath = string

type TrackList struct {
	gtk.ScrolledWindow
	parent ParentController

	Tree   *gtk.TreeView
	Store  *gtk.ListStore
	Select *gtk.TreeSelection

	Playlist  *state.Playlist
	TrackRows map[*state.Track]*TrackRow

	playing *state.Track
}

var trackListDragTargets = []gtk.TargetEntry{
	targetEntry("text/uri-list", gtk.TARGET_OTHER_APP, 1),
}

func targetEntry(target string, f gtk.TargetFlags, info uint) gtk.TargetEntry {
	e, _ := gtk.TargetEntryNew(target, f, info)
	return *e
}

func NewTrackList(parent ParentController, pl *state.Playlist) *TrackList {
	store, _ := gtk.ListStoreNew(
		glib.TYPE_STRING, // columnTitle
		glib.TYPE_STRING, // columnArtist
		glib.TYPE_STRING, // columnAlbum
		glib.TYPE_STRING, // columnTime
		glib.TYPE_INT,    // columnSelected - pango.Weight
	)

	tree, _ := gtk.TreeViewNewWithModel(store)
	tree.SetActivateOnSingleClick(false)
	tree.AppendColumn(newColumn("Title", columnTitle))
	tree.AppendColumn(newColumn("Artist", columnArtist))
	tree.AppendColumn(newColumn("Album", columnAlbum))
	tree.AppendColumn(newColumn("", columnTime))
	tree.AppendColumn(newColumn("", columnSelected))
	tree.Show()

	// tree.SetReorderable(true)
	// tree.Connect("")

	s, _ := tree.GetSelection()
	s.SetMode(gtk.SELECTION_MULTIPLE)

	scroll, _ := gtk.ScrolledWindowNew(nil, nil)
	scroll.SetPolicy(gtk.POLICY_NEVER, gtk.POLICY_AUTOMATIC)
	scroll.SetVExpand(true)
	scroll.Add(tree)
	scroll.Show()

	list := TrackList{
		ScrolledWindow: *scroll,
		parent:         parent,

		Tree:   tree,
		Store:  store,
		Select: s,

		Playlist:  pl,
		TrackRows: make(map[*state.Track]*TrackRow, len(pl.Tracks)),
	}

	// Create a nil slice and leave the allocation for later.
	var probeQueue []prober.Job

	for _, track := range pl.Tracks {
		row := &TrackRow{
			Iter: list.Store.Append(),
			Bold: false,
		}

		row.setListStore(track, list.Store)
		list.TrackRows[track] = row

		if !track.Metadata().IsProbed() {
			// Allocate a slice only if we have at least 1.
			if probeQueue == nil {
				probeQueue = make([]prober.Job, 0, len(pl.Tracks))
			}

			track := track // copy pointer

			job := prober.NewJob(track, func() {
				row.setListStore(track, list.Store)
				pl.SetUnsaved()
			})

			probeQueue = append(probeQueue, job)
		}
	}

	// TODO: this has a cache stampede problem. We need to have a context to
	// cancel this.
	prober.Queue(probeQueue...)

	tree.Connect("row-activated", func(_ *gtk.TextView, path *gtk.TreePath) {
		parent.PlayTrack(pl, path.GetIndices()[0])
	})

	// TODO: Implement TreeView drag-and-drop for reordering. Effectively, the
	// application should use the standardized file URI as the data for
	// reordering, which should make it work with a range of applications.
	//
	// To deal with the lack of data when we move the tracks out and in the
	// list, we could have a remove track and add path functions. The add
	// function would simply treat the track as an unprocessed one.

	tree.EnableModelDragDest(trackListDragTargets, gdk.ACTION_LINK)
	tree.Connect("drag-data-received",
		func(_ gtk.IWidget, ctx *gdk.DragContext, x, y int, data *gtk.SelectionData) {
			path, pos, ok := tree.GetDestRowAtPos(x, y)
			if !ok {
				log.Println("No path found at dragged pos.")
				return
			}

			// Get the files in form of line-delimited URIs
			var uris string
			if data.GetLength() > 0 {
				uris = string(data.GetData())
			}

			var paths = parseURIList(uris)

			if len(paths) > 0 {
				before := false ||
					pos == gtk.TREE_VIEW_DROP_BEFORE ||
					pos == gtk.TREE_VIEW_DROP_INTO_OR_BEFORE
				list.addTracksAt(path, before, paths)
			}
		},
	)

	tree.Connect("key-press-event", func(_ gtk.IWidget, ev *gdk.Event) bool {
		kp := gdk.EventKeyNewFromEvent(ev)

		switch kp.KeyVal() {
		case gdk.KEY_Delete:
			list.removeSelected()
			return true
		default:
			return false
		}
	})

	return &list
}

func parseURIList(list string) []string {
	// Get the files in form of line-delimited URIs
	var uris = strings.Fields(list)

	// Create a path slice that we decode URIs into.
	var paths = uris[:0]

	// Decode the URIs.
	for _, uri := range uris {
		u, err := url.Parse(uri)
		if err != nil {
			log.Printf("Failed parsing URI %q: %v\n", uri, err)
			continue
		}
		if u.Scheme != "file" {
			log.Println("Unknown file scheme (only locals):", u.Scheme)
			continue
		}
		paths = append(paths, u.Path)
	}

	return paths
}

func (list *TrackList) addTracksAt(path *gtk.TreePath, before bool, paths []string) {
	ix := path.GetIndices()[0]

	start, end := list.Playlist.Add(ix, before, paths...)
	probeQueue := make([]prober.Job, 0, end-start)

	for i := start; i < end; i++ {
		row := &TrackRow{
			Iter: list.Store.Insert(i),
			Bold: false,
		}
		track := list.Playlist.Tracks[i]

		row.setListStore(track, list.Store)
		list.TrackRows[track] = row

		job := prober.NewJob(track, func() {
			row.setListStore(track, list.Store)
		})

		probeQueue = append(probeQueue, job)
	}

	list.parent.UpdateTracks(list.Playlist)
	prober.Queue(probeQueue...)
}

func (list *TrackList) removeSelected() {
	selected := list.Select.GetSelectedRows(list.Store)
	selectIx := make([]int, 0, selected.Length())

	selected.Foreach(func(v interface{}) {
		path := v.(*gtk.TreePath)
		// Only count valid iters. This should most of the time be valid, but we
		// want to be sure.
		if iter, err := list.Store.GetIter(path); err == nil {
			list.Store.Remove(iter)
			selectIx = append(selectIx, path.GetIndices()[0])
		}
	})

	if len(selectIx) == 0 {
		return
	}

	list.Playlist.Remove(selectIx...)
	list.parent.UpdateTracks(list.Playlist)
}

// SetPlaying unbolds the last track (if any) and bolds the given track. It does
// not trigger any callback.
func (list *TrackList) SetPlaying(playing *state.Track) {
	rw, ok := list.TrackRows[playing]
	if !ok {
		log.Printf("Track not found on (*Tracklist).SetPlaying: %q\n", playing.Filepath)
		return
	}

	if list.playing != nil {
		playingRow := list.TrackRows[list.playing]
		playingRow.SetBold(list.Store, false)

		selectedRows := list.Select.CountSelectedRows()

		// Decide if we should move the selection.
		reselect := selectedRows == 0 ||
			(selectedRows == 1 && list.Select.IterIsSelected(playingRow.Iter))

		if reselect {
			list.Select.UnselectIter(playingRow.Iter)
			list.Select.SelectIter(rw.Iter)

			path, _ := list.Store.GetPath(rw.Iter)
			list.Tree.ScrollToCell(path, nil, false, 0, 0)
		}
	}

	list.playing = playing
	rw.SetBold(list.Store, true)
}
