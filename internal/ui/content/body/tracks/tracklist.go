package tracks

import (
	"log"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/diamondburned/aqours/internal/state"
	"github.com/diamondburned/aqours/internal/state/prober"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
	"github.com/lithammer/fuzzysearch/fuzzy"
	"github.com/pkg/errors"
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

	menu *gtk.Menu
}

var trackListDragTargets = []gtk.TargetEntry{
	targetEntry("text/uri-list", gtk.TARGET_OTHER_APP, 1),
}

func targetEntry(target string, f gtk.TargetFlags, info uint) gtk.TargetEntry {
	e, _ := gtk.TargetEntryNew(target, f, info)
	return *e
}

// searchFuzzy implements TreeViewSearchEqualFunc.
func searchFuzzy(m *gtk.TreeModel, col int, k string, it *gtk.TreeIter) bool {
	return !fuzzy.MatchNormalizedFold(k, m.GetStringFromIter(it))
}

type columnType = int

const (
	columnTitle columnType = iota
	columnArtist
	columnAlbum
	columnTime
	columnSelected
	columnSearchData
)

const maxDataSize = 10 * 1024 * 1024 // 10MB

func NewTrackList(parent ParentController, pl *state.Playlist) *TrackList {
	store, _ := gtk.ListStoreNew(
		glib.TYPE_STRING, // columnTitle
		glib.TYPE_STRING, // columnArtist
		glib.TYPE_STRING, // columnAlbum
		glib.TYPE_STRING, // columnTime
		glib.TYPE_INT,    // columnSelected - pango.Weight
		glib.TYPE_STRING, // columnSearchData
	)

	tree, _ := gtk.TreeViewNewWithModel(store)
	tree.SetActivateOnSingleClick(false)
	tree.SetProperty("has-tooltip", true)
	tree.AppendColumn(newColumn("Title", columnTitle))
	tree.AppendColumn(newColumn("Artist", columnArtist))
	tree.AppendColumn(newColumn("Album", columnAlbum))
	tree.AppendColumn(newColumn("", columnTime))
	tree.AppendColumn(newColumn("", columnSelected))
	tree.AppendColumn(newColumn("", columnSearchData))
	tree.SetSearchColumn(columnSearchData)
	tree.SetSearchEqualFunc(searchFuzzy)
	tree.Show()

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
			if dataLen := data.GetLength(); dataLen < 1 || dataLen > maxDataSize {
				return
			}
			// Get the files in form of line-delimited URIs
			var uris = string(data.GetData())

			path, pos, ok := tree.GetDestRowAtPos(x, y)
			if !ok {
				log.Println("No path found at dragged pos.")
				return
			}

			before := false ||
				pos == gtk.TREE_VIEW_DROP_BEFORE ||
				pos == gtk.TREE_VIEW_DROP_INTO_OR_BEFORE

			tree.SetSensitive(false)

			go func() {
				var paths = parseURIList(uris)

				glib.IdleAdd(func() {
					tree.SetSensitive(true)
					if len(paths) > 0 {
						list.addTracksAt(path, before, paths)
					}
				})
			}()
		},
	)

	tree.Connect("key-press-event", func(_ gtk.IWidget, ev *gdk.Event) bool {
		return list.handleKeyEvent(gdk.EventKeyNewFromEvent(ev))
	})

	refresh, _ := gtk.MenuItemNewWithLabel("Refresh Metadata")
	refresh.Show()
	refresh.Connect("activate", list.refreshSelected)

	sortTracks, _ := gtk.MenuItemNewWithLabel("Sort")
	sortTracks.Show()
	sortTracks.Connect("activate", list.sortSelectedTracks)

	list.menu, _ = gtk.MenuNew()
	list.menu.Add(refresh)
	list.menu.Add(sortTracks)

	tree.Connect("button-press-event", func(_ gtk.IWidget, ev *gdk.Event) bool {
		bp := gdk.EventButtonNewFromEvent(ev)

		switch bp.Button() {
		case gdk.BUTTON_SECONDARY:
			list.menu.PopupAtPointer(ev)
			return true
		default:
			return false
		}
	})

	// This leaks one Pixbuf but who cares.
	trackTooltip := newTrackTooltipBox()

	tree.Connect("query-tooltip",
		func(_ gtk.IWidget, x, y int, kb bool, t *gtk.Tooltip) bool {
			var path *gtk.TreePath
			if !kb {
				path, _, _ = tree.GetDestRowAtPos(x, y)
			} else {
				_, iter, _ := list.Select.GetSelected()
				if iter != nil {
					path, _ = list.Store.GetPath(iter)
				}
			}

			if path == nil {
				return false
			}

			ix := path.GetIndices()[0]
			trackTooltip.Attach(t, list.Playlist.Tracks[ix])

			return true
		},
	)

	return &list
}

func parseURIList(list string) []string {
	// Get the files in form of line-delimited URIs
	var uris = strings.Fields(list)

	// Create a path slice that we decode URIs into.
	var paths = make([]string, 0, len(uris))

	// Decode the URIs.
	for _, uri := range uris {
		u, err := url.Parse(uri)
		if err != nil {
			log.Printf("Failed parsing URI %q: %v\n", uri, err)
			continue
		}
		if u.Scheme != "file" && u.Scheme != "" {
			log.Printf("Unknown file URI scheme (only locals): %q\n", uri)
			continue
		}

		if err := readDirOrFile(u.Path, &paths); err != nil {
			log.Printf("Failed to read %q: %v\n", u.Path, err)
		}
	}

	return paths
}

func readDirOrFile(path string, dest *[]string) error {
	f, err := os.Open(path)
	if err != nil {
		return errors.Wrap(err, "failed to open file")
	}
	defer f.Close()

	s, err := f.Stat()
	if err != nil {
		return errors.Wrap(err, "file stat failed while drag-and-drop")
	}

	if !s.IsDir() {
		*dest = append(*dest, path)
		return nil
	}

	files, err := f.Readdirnames(0)
	if err != nil {
		return errors.Wrap(err, "failed to read dir names")
	}

	for _, file := range files {
		*dest = append(*dest, filepath.Join(path, file))
	}

	return nil
}

func (list *TrackList) handleKeyEvent(evk *gdk.EventKey) bool {
	keyVal := evk.KeyVal()
	keyMod := gdk.ModifierType(evk.State())

	switch keyVal {
	case gdk.KEY_Delete:
		list.removeSelected()
		return true
	}

	if modIsPressed(keyMod, gdk.CONTROL_MASK) {
		switch keyVal {
		case gdk.KEY_S: // Ctrl+S
			list.parent.SavePlaylist(list.Playlist)
			return true
		}
	}

	return false
}

func modIsPressed(mod, press gdk.ModifierType) bool {
	return mod&press == press
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
		selectIx = append(selectIx, path.GetIndices()[0])
	})

	if len(selectIx) == 0 {
		return
	}

	for _, ix := range selectIx {
		track := list.Playlist.Tracks[ix]
		trRow := list.TrackRows[track]

		delete(list.TrackRows, track)
		list.Store.Remove(trRow.Iter)
	}

	list.Playlist.Remove(selectIx...)
	list.parent.UpdateTracks(list.Playlist)
}

func (list *TrackList) sortSelectedTracks() {
	selected := list.Select.GetSelectedRows(list.Store)
	selectMin := -1
	selectMax := -1

	selected.Foreach(func(v interface{}) {
		path := v.(*gtk.TreePath)
		ix := path.GetIndices()[0]

		// Get max and min bounds without allocating a slice.
		if selectMin == -1 || ix < selectMin {
			selectMin = ix
		}
		if selectMax == -1 || ix > selectMax {
			selectMax = ix
		}
	})

	list.sortTracksBounds(selectMin, selectMax)
}

func (list *TrackList) sortTracksBounds(start, end int) {
	// Exit if we have nothing.
	if start == end {
		return
	}

	sorter := newTrackSorter(list, start, end)
	sort.Stable(sorter)
	list.parent.UpdateTracks(list.Playlist)
}

func (list *TrackList) refreshSelected() {
	selected := list.Select.GetSelectedRows(list.Store)
	selectIx := make([]int, 0, selected.Length())

	selected.Foreach(func(v interface{}) {
		path := v.(*gtk.TreePath)
		selectIx = append(selectIx, path.GetIndices()[0])
	})

	if len(selectIx) == 0 {
		return
	}

	probeQueue := make([]prober.Job, len(selectIx))

	for i, ix := range selectIx {
		track := list.Playlist.Tracks[ix]
		probeQueue[i] = prober.NewJob(track, func() {
			row := list.TrackRows[track]
			row.setListStore(track, list.Store)
		})
	}

	prober.Queue(probeQueue...)
}

func (list *TrackList) SelectPlaying() {
	rw, ok := list.TrackRows[list.playing]
	if !ok {
		return
	}

	list.Select.UnselectAll()
	list.Select.SelectIter(rw.Iter)

	path, _ := list.Store.GetPath(rw.Iter)
	list.Tree.ScrollToCell(path, nil, false, 0, 0)
}

// SetPlaying unbolds the last track (if any) and bolds the given track. It does
// not trigger any callback.
func (list *TrackList) SetPlaying(playing *state.Track) {
	rw, ok := list.TrackRows[playing]
	if !ok {
		log.Printf("Track not found on (*Tracklist).SetPlaying: %q\n", playing.Filepath)
		return
	}

	// If we have nothing playing, then we should reselect. I'm not sure of this
	// behavior.
	reselect := list.playing == nil

	if list.playing != nil {
		playingRow := list.TrackRows[list.playing]
		playingRow.SetBold(list.Store, false)

		// Decide if we should move the selection.
		selectedRows := list.Select.CountSelectedRows()
		reselect = selectedRows == 1 && list.Select.IterIsSelected(playingRow.Iter)

		if reselect {
			list.Select.UnselectIter(playingRow.Iter)
		}
	}

	if reselect {
		list.Select.SelectIter(rw.Iter)

		path, _ := list.Store.GetPath(rw.Iter)
		list.Tree.ScrollToCell(path, nil, false, 0, 0)
	}

	list.playing = playing
	rw.SetBold(list.Store, true)
}
