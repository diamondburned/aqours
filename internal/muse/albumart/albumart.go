package albumart

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dhowden/tag"
)

// Stolen from: mpv/blob/master/player/external_files.c#L45, which was
// stolen from: vlc/blob/master/modules/meta_engine/folder.c#L40.
// Sorted by priority.
var coverFiles = []string{
	"AlbumArt.jpg",
	"Album.jpg",
	"cover.jpg",
	"cover.png",
	"front.jpg",
	"front.png",
	"Cover.jpg",

	"AlbumArtSmall.jpg",
	"Folder.jpg",
	"Folder.png",
	".folder.png",
	"thumb.jpg",

	"front.bmp",
	"front.gif",
	"cover.gif",
}

type File struct {
	io.ReadCloser
	Extension string // jpeg, ...
}

func (f File) IsValid() bool {
	return f.ReadCloser != nil
}

// AlbumArt queries for an album art. It returns an invalid File if there is no
// album art. The function may read the album art into memory.
func AlbumArt(path string) File {
	// Prioritize searching for external album arts over reading the album art
	// into memory.
	dir := filepath.Dir(path)

	for _, coverFile := range coverFiles {
		f, err := os.Open(filepath.Join(dir, coverFile))
		if err != nil {
			continue
		}

		return File{
			ReadCloser: f,
			Extension:  normalizeExt(filepath.Ext(coverFile)),
		}
	}

	f, err := os.Open(path)
	if err != nil {
		return File{}
	}
	defer f.Close()

	// Use a 1 minute timeout.
	f.SetDeadline(time.Now().Add(time.Minute))

	m, err := tag.ReadFrom(f)
	if err == nil {
		if pic := m.Picture(); pic != nil {
			return File{
				ReadCloser: ioutil.NopCloser(bytes.NewReader(pic.Data)),
				Extension:  normalizeExt(pic.Ext),
			}
		}
	}

	return File{}
}

func normalizeExt(ext string) string {
	ext = strings.TrimPrefix(ext, ".")
	ext = strings.ToLower(ext)

	if ext == "jpg" {
		ext = "jpeg"
	}

	return ext
}
