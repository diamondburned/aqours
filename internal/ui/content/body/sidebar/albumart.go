package sidebar

import (
	"context"
	"io"
	"log"
	"time"

	"github.com/diamondburned/aqours/internal/muse/albumart"
	"github.com/diamondburned/aqours/internal/state"
	"github.com/diamondburned/aqours/internal/ui/css"
	"github.com/diamondburned/gotk4/pkg/core/gioutil"
	"github.com/diamondburned/gotk4/pkg/gdkpixbuf/v2"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

var albumArtCSS = css.PrepareClass("album-art", "")

type AlbumArt struct {
	*gtk.Revealer
	Path  string
	Image *gtk.Image

	stopLoading context.CancelFunc
}

const AlbumArtSize = 192

func NewAlbumArt() *AlbumArt {
	img := gtk.NewImage()
	img.SetSizeRequest(AlbumArtSize, AlbumArtSize)
	img.SetIconSize(gtk.IconSizeLarge)
	img.SetVAlign(gtk.AlignCenter)
	img.SetHAlign(gtk.AlignCenter)

	albumArtCSS(img)

	rev := gtk.NewRevealer()
	rev.SetRevealChild(true)
	rev.SetTransitionType(gtk.RevealerTransitionTypeSlideUp)
	rev.SetChild(img)

	aa := &AlbumArt{
		Revealer:    rev,
		Image:       img,
		stopLoading: func() {}, // stub
	}

	aa.SetTrack(nil)

	return aa
}

func (aa *AlbumArt) SetTrack(track *state.Track) {
	aa.Image.SetFromIconName("media-optical-symbolic")

	if track == nil {
		return
	}

	aa.stopLoading()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	aa.stopLoading = cancel

	aa.Path = track.Filepath
	scale := aa.Image.ScaleFactor()

	go func() {
		defer cancel()

		pixbuf := FetchAlbumArtScaled(ctx, track, AlbumArtSize, scale)
		if pixbuf == nil {
			return
		}

		glib.IdleAdd(func() {
			// Make sure that the album art is still displaying the same file.
			if aa.Path == track.Filepath {
				aa.Image.SetFromPixbuf(pixbuf)
			}
		})
	}()
}

// FetchAlbumArtScaled fetches the track's album art into a scaled Cairo Surface
// for HiDPI.
func FetchAlbumArtScaled(ctx context.Context, track *state.Track, size, scale int) *gdkpixbuf.Pixbuf {
	return FetchAlbumArt(ctx, track, size*scale)
}

// FetchAlbumArt fetches the track's album art into a pixbuf with the given
// size.
func FetchAlbumArt(ctx context.Context, track *state.Track, size int) *gdkpixbuf.Pixbuf {
	f := albumart.AlbumArt(ctx, track.Filepath)
	if f == nil {
		return nil
	}
	defer f.Close()

	l, err := gdkpixbuf.NewPixbufLoaderWithType(f.Extension)
	if err != nil {
		log.Printf("PixbufLoaderNewWithType failed with %q: %v\n", f.Extension, err)
		return nil
	}
	defer l.Close()

	l.ConnectSizePrepared(func(w, h int) {
		l.SetSize(MaxSize(w, h, size, size))
	})

	// Trivial error that we can't handle.
	if _, err := io.Copy(gioutil.PixbufLoaderWriter(l), f.ReadCloser); err != nil {
		log.Println("PixbufLoader.Write:", err)
		return nil
	}

	if err := l.Close(); err != nil {
		log.Println("PixbufLoader.Close:", err)
		return nil
	}

	return l.Pixbuf()
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
