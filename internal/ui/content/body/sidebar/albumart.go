package sidebar

import (
	"io"
	"log"

	"github.com/diamondburned/aqours/internal/muse/albumart"
	"github.com/diamondburned/aqours/internal/state"
	"github.com/diamondburned/aqours/internal/ui/css"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
)

var albumArtCSS = css.PrepareClass("album-art", "")

type AlbumArt struct {
	gtk.Revealer
	Path  string
	Image *gtk.Image
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
		Revealer: *rev,
		Image:    img,
	}

	aa.SetTrack(nil)

	return aa
}

func (aa *AlbumArt) SetTrack(track *state.Track) {
	aa.Image.SetFromIconName("media-optical-symbolic", gtk.ICON_SIZE_DIALOG)

	if track == nil {
		return
	}

	aa.Path = track.Filepath

	go func() {
		p := FetchAlbumArt(track, AlbumArtSize)
		if p == nil {
			return
		}

		glib.IdleAdd(func() {
			// Make sure that the album art is still displaying the same file.
			if aa.Path == track.Filepath {
				aa.Image.SetFromPixbuf(p)
			}
		})
	}()
}

func FetchAlbumArt(track *state.Track, size int) *gdk.Pixbuf {
	var f = albumart.AlbumArt(track.Filepath)
	if !f.IsValid() {
		return nil
	}

	defer f.Close()

	l, err := gdk.PixbufLoaderNewWithType(f.Extension)
	if err != nil {
		log.Println("PixbufLoaderNewWithType failed with jpeg:", err)
		return nil
	}
	defer l.Close()

	l.Connect("size-prepared", func(l *gdk.PixbufLoader, w, h int) {
		l.SetSize(MaxSize(w, h, size, size))
	})

	if _, err := io.Copy(l, f.ReadCloser); err != nil {
		log.Println("Failed to write to pixbuf:", err)
		return nil
	}

	p, err := l.GetPixbuf()
	if err != nil {
		log.Println("Failed to get pixbuf:", err)
		// Allow setting a nil pixbuf if we have an error.
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
