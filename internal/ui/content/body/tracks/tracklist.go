package tracks

import (
	"log"
	"net/url"
	"runtime"
	"strings"
	"sync"

	"github.com/diamondburned/aqours/internal/muse/playlist"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
)

type TrackPath = string

type TrackList struct {
	gtk.ScrolledWindow
	Tree   *gtk.TreeView
	Store  *gtk.ListStore
	Select *gtk.TreeSelection

	Tracks []*Track

	playing *Track
}

var trackListDragTargets = []gtk.TargetEntry{
	targetEntry("text/uri-list", gtk.TARGET_OTHER_APP, 1),
}

func targetEntry(target string, f gtk.TargetFlags, info uint) gtk.TargetEntry {
	e, _ := gtk.TargetEntryNew(target, f, info)
	return *e
}

func NewTrackList(parent ParentController, pl *playlist.Playlist) *TrackList {
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

	tree.Connect("row-activated", func(_ *gtk.TextView, path *gtk.TreePath) {
		parent.PlayTrack(pl.Name, path.GetIndices()[0])
	})

	// tree.SetReorderable(true)
	// tree.Connect("")

	tree.DragDestSet(gtk.DEST_DEFAULT_ALL, trackListDragTargets, gdk.ACTION_LINK)
	tree.Connect("drag-data-received",
		func(_ gtk.IWidget, ctx *gdk.DragContext, x, y uint, data *gtk.SelectionData) {
			// Get the files in form of line-delimited URIs
			var uris = strings.Fields(string(data.GetData()))

			// Create a path slice that we decode URIs into.
			var paths = uris[:0]

			// Decode the URIs.
			for _, uri := range uris {
				u, err := url.Parse(uri)
				if err != nil {
					log.Printf("Failed parsing URI %q: %v\n", uri, err)
					continue
				}
				paths = append(paths, u.Path)
			}

			log.Printf("%q\n", paths)
		},
	)

	s, _ := tree.GetSelection()
	s.SetMode(gtk.SELECTION_MULTIPLE)

	scroll, _ := gtk.ScrolledWindowNew(nil, nil)
	scroll.SetPolicy(gtk.POLICY_NEVER, gtk.POLICY_AUTOMATIC)
	scroll.SetVExpand(true)
	scroll.Add(tree)
	scroll.Show()

	list := TrackList{
		ScrolledWindow: *scroll,

		Tree:   tree,
		Store:  store,
		Select: s,

		Tracks: make([]*Track, len(pl.Tracks)),
	}

	var probeQueue = make([]*probeJob, 0, len(pl.Tracks))

	for i, track := range pl.Tracks {
		advTrack := &Track{
			Track: track,
			Iter:  list.Store.Append(),
		}

		advTrack.setListStore(list.Store)
		list.Tracks[i] = advTrack

		if !track.IsProbed() {
			probeQueue = append(probeQueue, &probeJob{
				adv:   advTrack,
				track: *track,
			})
		}
	}

	// TODO: this has a cache stampede problem. We need to have a context to
	// cancel this.
	go batchProbe(list.Store, probeQueue)

	return &list
}

func (list *TrackList) findTrack(track *playlist.Track) *Track {
	for _, advTrack := range list.Tracks {
		if advTrack.Track == track {
			return advTrack
		}
	}
	return nil
}

func (list *TrackList) SetPlaying(playing *playlist.Track) {
	var advTrack = list.findTrack(playing)

	if list.playing != nil {
		list.playing.SetBold(list.Store, false)

		// Decide if we should move the selection.
		reselect := true &&
			list.Select.CountSelectedRows() == 1 &&
			list.Select.IterIsSelected(list.playing.Iter)

		if reselect {
			list.Select.UnselectIter(list.playing.Iter)
			list.Select.SelectIter(advTrack.Iter)

			path, _ := list.Store.GetPath(advTrack.Iter)
			list.Tree.ScrollToCell(path, nil, false, 0, 0)
		}
	}

	list.playing = advTrack
	list.playing.SetBold(list.Store, true)
}

var maxJobs = runtime.GOMAXPROCS(-1)

// probeJob is an internal type that allows a track to be copied in a
// thread-safe way for probing.
type probeJob struct {
	adv   *Track
	track playlist.Track
}

// batchProbe batch probes the given slice of track pointers. Although a slice
// of pointers are given, the Probe method will actually be called on a copy of
// the track. The probed callback should therefore reapply the track.
func batchProbe(store ListStorer, tracks []*probeJob) {
	queue := make(chan *probeJob, maxJobs)
	waitg := sync.WaitGroup{}
	waitg.Add(maxJobs)

	for i := 0; i < maxJobs; i++ {
		go func() {
			defer waitg.Done()

			for job := range queue {
				job := job // copy must

				if err := job.track.Probe(); err != nil {
					log.Printf("Failed to probe %q: %v", job.track.Filepath, err)
					continue
				}

				glib.IdleAdd(func() {
					// Update everything inside the old track.
					*job.adv.Track = job.track

					// Update the list entry afterwards.
					// TODO: check invalidation.
					job.adv.setListStore(store)
				})
			}
		}()
	}

	for _, track := range tracks {
		queue <- track
	}

	close(queue)
	waitg.Wait()
}
