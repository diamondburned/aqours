package sidebar

import (
	"context"
	"io"
	"log"
	"time"

	"github.com/diamondburned/aqours/internal/muse/albumart"
	"github.com/diamondburned/aqours/internal/state"
	"github.com/diamondburned/aqours/internal/ui/css"
	"github.com/gotk3/gotk3/cairo"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
)

var albumArtCSS = css.PrepareClass("album-art", "")

type AlbumArt struct {
	gtk.Revealer
	Path  string
	Image *gtk.Image

	stopLoading context.CancelFunc
}

const AlbumArtSize = 192

func NewAlbumArt() *AlbumArt {
	img, _ := gtk.ImageNew()
	img.SetSizeRequest(AlbumArtSize, AlbumArtSize)
	img.SetVAlign(gtk.ALIGN_CENTER)
	img.SetHAlign(gtk.ALIGN_CENTER)
	img.Show()
	albumArtCSS(img)

	rev, _ := gtk.RevealerNew()
	rev.SetRevealChild(true)
	rev.SetTransitionType(gtk.REVEALER_TRANSITION_TYPE_SLIDE_UP)
	rev.Add(img)
	rev.Show()

	aa := &AlbumArt{
		Revealer:    *rev,
		Image:       img,
		stopLoading: func() {}, // stub
	}

	aa.SetTrack(nil)

	return aa
}

func (aa *AlbumArt) SetTrack(track *state.Track) {
	aa.Image.SetFromIconName("media-optical-symbolic", gtk.ICON_SIZE_DIALOG)

	if track == nil {
		return
	}

	aa.stopLoading()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	aa.stopLoading = cancel

	aa.Path = track.Filepath
	scale := aa.Image.GetScaleFactor()

	go func() {
		defer cancel()

		surface := FetchAlbumArtScaled(ctx, track, AlbumArtSize, scale)
		if surface == nil {
			return
		}

		glib.IdleAdd(func() {
			// Make sure that the album art is still displaying the same file.
			if aa.Path == track.Filepath {
				aa.Image.SetFromSurface(surface)
			}
		})
	}()
}

// FetchAlbumArtScaled fetches the track's album art into a scaled Cairo Surface
// for HiDPI.
func FetchAlbumArtScaled(ctx context.Context, track *state.Track, size, scale int) *cairo.Surface {
	p := FetchAlbumArt(ctx, track, size*scale)
	if p == nil {
		return nil
	}

	c, err := gdk.CairoSurfaceCreateFromPixbuf(p, scale, nil)
	if err != nil {
		log.Println("Failed to get Cairo Surface from Pixbuf:", err)
		return nil
	}

	return c
}

// FetchAlbumArt fetches the track's album art into a pixbuf with the given
// size.
func FetchAlbumArt(ctx context.Context, track *state.Track, size int) *gdk.Pixbuf {
	var f = albumart.AlbumArt(ctx, track.Filepath)
	if !f.IsValid() {
		return nil
	}

	defer f.Close()

	l, err := gdk.PixbufLoaderNewWithType(f.Extension)
	if err != nil {
		log.Printf("PixbufLoaderNewWithType failed with %q: %v\n", f.Extension, err)
		return nil
	}
	defer l.Close()

	l.Connect("size-prepared", func(l *gdk.PixbufLoader, w, h int) {
		l.SetSize(MaxSize(w, h, size, size))
	})

	// Trivial error that we can't handle.
	if _, err := io.Copy(l, f.ReadCloser); err != nil {
		return nil
	}

	p, err := l.GetPixbuf()
	if err != nil {
		log.Println("Failed to get pixbuf:", err)
		return nil
	}

	return p
}

// MaxSize returns the maximum size that can fit within the given max width and
// height. Aspect ratio is preserved.
func MaxSize(w, h, maxW, maxH int) (int, int) {
	if w < maxW && h < maxH {
		return w, h
	}

	if w > h {
		h = h * maxW / w
		w = maxW
	} else {
		w = w * maxH / h
		h = maxH
	}

	return w, h
}
