package tracks

import (
	"context"
	"fmt"
	"html"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/diamondburned/aqours/internal/durafmt"
	"github.com/diamondburned/aqours/internal/state"
	"github.com/diamondburned/aqours/internal/ui/content/body/sidebar"
	"github.com/diamondburned/aqours/internal/ui/css"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
	"github.com/gotk3/gotk3/pango"
)

type ParentController interface {
	gtk.IWindow
	PlayTrack(p *state.Playlist, index int)
	SavePlaylist(p *state.Playlist)
	UpdateTracks(p *state.Playlist)
}

type ListStorer interface {
	Set(iter *gtk.TreeIter, columns []int, values []interface{}) error
	SetValue(iter *gtk.TreeIter, column int, value interface{}) error
}

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
	Iter *gtk.TreeIter
}

func (row *TrackRow) SetBold(store ListStorer, bold bool) {
	row.Bold = bold
	store.SetValue(row.Iter, columnSelected, weight(row.Bold))
}

func (row *TrackRow) setListStore(t *state.Track, store ListStorer) {
	metadata := t.Metadata()

	searchData := strings.Builder{}
	searchData.WriteString(metadata.Title)
	searchData.WriteByte(' ')
	searchData.WriteString(metadata.Artist)
	searchData.WriteByte(' ')
	searchData.WriteString(metadata.Album)
	searchData.WriteByte(' ')
	searchData.WriteString(metadata.Filepath)

	store.Set(
		row.Iter,
		[]int{
			columnTitle,
			columnArtist,
			columnAlbum,
			columnTime,
			columnSelected,
			columnSearchData,
		},
		[]interface{}{
			metadata.Title,
			metadata.Artist,
			metadata.Album,
			durafmt.Format(metadata.Length),
			weight(row.Bold),
			searchData.String(),
		},
	)
}

func weight(bold bool) pango.Weight {
	if bold {
		return pango.WEIGHT_BOLD
	}
	return pango.WEIGHT_BOOK
}

const (
	AlbumIconSize = gtk.ICON_SIZE_DIALOG
	PixelIconSize = 96
)

var trackTooltipCSS = css.PrepareClass("track-tooltip", "")

var trackTooltipImageCSS = css.PrepareClass("track-tooltip-image", `
	image {
		margin: 6px;
	}
`)

type trackTooltipBox struct {
	image     *gdk.Pixbuf
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

	if tt.image != nil {
		t.SetIcon(tt.image)
	} else {
		t.SetIconFromIconName("folder-music-symbolic", AlbumIconSize)

		// Gtk is very, VERY dumb, so it'll try and fetch an album art just by
		// hovering the cursor on things. I simply cannot fix stupidity of this
		// level, so just deal with the fd spam. At least I have a context to
		// cancel things.
		if newTrack {
			tt.stopFetch()
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			tt.stopFetch = cancel

			go func() {
				defer cancel()

				p := sidebar.FetchAlbumArt(ctx, track, PixelIconSize)
				if p == nil {
					return
				}

				glib.IdleAdd(func() {
					if tt.trackPath == mdata.Filepath {
						tt.image = p
						t.SetIcon(p)
					}
				})
			}()
		}
	}

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
