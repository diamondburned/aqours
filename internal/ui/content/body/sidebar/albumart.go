package sidebar

import (
	"log"

	"github.com/diamondburned/aqours/internal/muse/playlist"
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

func (aa *AlbumArt) SetTrack(track *playlist.Track) {
	aa.Image.SetFromIconName("media-optical-symbolic", gtk.ICON_SIZE_DIALOG)

	if track == nil {
		return
	}

	aa.Path = track.Filepath

	go func() {
		a, err := track.AlbumArt()
		if a == nil {
			if err != nil {
				log.Println("Failed to get album art:", err)
			}
			return
		}

		// We need to do this to make GdkPixbufLoader happy.
		if a.Ext == "jpg" {
			a.Ext = "jpeg"
		}

		l, err := gdk.PixbufLoaderNewWithType(a.Ext)
		if err != nil {
			log.Println("PixbufLoaderNewWithType failed with jpeg:", err)
			return
		}
		defer l.Close()

		l.Connect("size-prepared", func(l *gdk.PixbufLoader, w, h int) {
			l.SetSize(MaxSize(w, h, AlbumArtSize, AlbumArtSize))
		})

		p, err := l.WriteAndReturnPixbuf(a.Data)
		if err != nil {
			log.Println("Failed to write to puxbuf:", err)
		}

		glib.IdleAdd(func() {
			// Make sure that the album art is still displaying the same file.
			if aa.Path == track.Filepath {
				aa.Image.SetFromPixbuf(p)
			}
		})
	}()
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
