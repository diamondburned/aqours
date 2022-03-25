package tracks

import (
	"context"
	"fmt"
	"html"
	"io"
	"strconv"
	"strings"

	"github.com/diamondburned/aqours/internal/durafmt"
	"github.com/diamondburned/aqours/internal/state"
	"github.com/diamondburned/aqours/internal/ui/css"
	"github.com/diamondburned/gotk4/pkg/gdkpixbuf/v2"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/diamondburned/gotk4/pkg/pango"
)

type ParentController interface {
	PlayTrack(p *state.Playlist, index int)
	SavePlaylist(p *state.Playlist)
	UpdateTracks(p *state.Playlist)
}

type Container struct {
	gtk.Stack
	parent ParentController

	Lists map[string]*TrackList // tree model

	// current treeview playlist name
	current string
}

func NewContainer(parent ParentController) *Container {
	stack := gtk.NewStack()
	stack.SetTransitionType(gtk.StackTransitionTypeCrossfade)
	stack.SetTransitionDuration(25)

	return &Container{
		parent: parent,
		Stack:  *stack,
		Lists:  map[string]*TrackList{},
	}
}

func (c *Container) SelectPlaylist(playlist *state.Playlist) *TrackList {
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
	r := gtk.NewCellRendererText()
	r.SetObjectProperty("weight-set", true)
	r.SetObjectProperty("ellipsize", pango.EllipsizeEnd)
	r.SetObjectProperty("ellipsize-set", true)

	c := gtk.NewTreeViewColumn()
	c.SetTitle(text)
	c.PackStart(r, false)
	c.AddAttribute(r, "text", int(col))
	c.AddAttribute(r, "weight", int(columnSelected))
	c.SetSizing(gtk.TreeViewColumnFixed)
	c.SetResizable(true)

	switch col {
	case columnTime:
		c.SetMinWidth(50)
	case columnSelected, columnSearchData:
		c.SetVisible(false)
	default:
		c.SetExpand(true)
		c.SetMinWidth(150)
	}

	return c
}

type TrackRow struct {
	Bold bool
	iter struct {
		path  *gtk.TreePath
		store *gtk.ListStore
	}
	// Path *gtk.TreePath
}

func (row *TrackRow) Iter() (*gtk.TreeIter, bool) {
	return row.iter.store.Iter(row.iter.path)
}

func (row *TrackRow) Path() *gtk.TreePath { return row.iter.path }

func (row *TrackRow) Remove() bool {
	if iter, ok := row.Iter(); ok {
		row.iter.store.Remove(iter)
		return true
	}
	return false
}

func (row *TrackRow) SetBold(store *gtk.ListStore, bold bool) {
	it, ok := row.Iter()
	if !ok {
		return
	}

	row.Bold = bold
	store.SetValue(it, columnSelected, glib.NewValue(weight(row.Bold)))
}

func (row *TrackRow) setListStore(t *state.Track) {
	it, ok := row.Iter()
	if !ok {
		return
	}

	metadata := t.Metadata()

	searchData := strings.Builder{}
	searchData.WriteString(metadata.Title)
	searchData.WriteByte(' ')
	searchData.WriteString(metadata.Artist)
	searchData.WriteByte(' ')
	searchData.WriteString(metadata.Album)
	searchData.WriteByte(' ')
	searchData.WriteString(metadata.Filepath)

	row.iter.store.Set(
		it,
		[]int{
			columnTitle,
			columnArtist,
			columnAlbum,
			columnTime,
			columnSelected,
			columnSearchData,
		},
		[]glib.Value{
			*glib.NewValue(metadata.Title),
			*glib.NewValue(metadata.Artist),
			*glib.NewValue(metadata.Album),
			*glib.NewValue(durafmt.Format(metadata.Length)),
			*glib.NewValue(weight(row.Bold)),
			*glib.NewValue(searchData.String()),
		},
	)
}

func weight(bold bool) pango.Weight {
	if bold {
		return pango.WeightBold
	}
	return pango.WeightBook
}

const (
	AlbumIconSize = gtk.IconSizeLarge
	PixelIconSize = 96
)

var trackTooltipCSS = css.PrepareClass("track-tooltip", "")

var trackTooltipImageCSS = css.PrepareClass("track-tooltip-image", `
	.track-tooltip-image {
		margin: 6px;
	}
`)

type trackTooltipBox struct {
	image     *gdkpixbuf.Pixbuf
	trackPath string
	stopFetch context.CancelFunc
}

func newTrackTooltipBox() *trackTooltipBox {
	return &trackTooltipBox{
		stopFetch: func() {}, // stub
	}
}

func (tt *trackTooltipBox) Attach(t *gtk.Tooltip, track *state.Track) {
	mdata := track.Metadata()

	newTrack := tt.trackPath != mdata.Filepath
	if newTrack {
		tt.trackPath = mdata.Filepath
		tt.image = nil
	}

	// TODO: album art fetchign is very expensive. Until we can reduce the call
	// frequency, we should absolutely not do this.

	// if tt.image != nil {
	// 	t.SetIcon(tt.image)
	// } else {
	// 	t.SetIconFromIconName("folder-music-symbolic", AlbumIconSize)

	// 	// Gtk is very, VERY dumb, so it'll try and fetch an album art just by
	// 	// hovering the cursor on things. I simply cannot fix stupidity of this
	// 	// level, so just deal with the fd spam. At least I have a context to
	// 	// cancel things.
	// 	if newTrack {
	// 		tt.stopFetch()
	// 		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	// 		tt.stopFetch = cancel

	// 		go func() {
	// 			defer cancel()

	// 			p := sidebar.FetchAlbumArt(ctx, track, PixelIconSize)
	// 			if p == nil {
	// 				return
	// 			}

	// 			glib.IdleAdd(func() {
	// 				if tt.trackPath == mdata.Filepath {
	// 					tt.image = p
	// 					t.SetIcon(p)
	// 				}
	// 			})
	// 		}()
	// 	}
	// }

	var builder strings.Builder
	writeHTMLField(&builder, "<b>Title:</b> %s\n", mdata.Title)
	writeHTMLField(&builder, "<b>Artist:</b> %s\n", mdata.Artist)
	writeHTMLField(&builder, "<b>Album:</b> %s\n", mdata.Album)
	writeHTMLField(&builder, "<b>Number:</b> %s\n", strconv.Itoa(mdata.Number))
	writeHTMLField(&builder, "<b>Length:</b> %s\n", durafmt.Format(mdata.Length))
	writeHTMLField(&builder,
		"<b>Filepath:</b> <span insert-hyphens=\"false\">%s</span>",
		mdata.Filepath,
	)

	t.SetMarkup(builder.String())
}

func writeHTMLField(w io.Writer, f, v string) {
	if v != "" {
		fmt.Fprintf(w, f, html.EscapeString(v))
	}
}
