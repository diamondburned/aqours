package albumart

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"

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
	"folder.jpg",
	".folder.png",
	"thumb.jpg",
	"Thumb.jpg",

	"front.bmp",
	"front.gif",
	"cover.gif",
}

type File struct {
	io.ReadCloser
	Extension string // jpeg, ...
}

// AlbumArt queries for an album art. It returns an invalid File if there is no
// album art. The function may read the album art into memory.
//
// The given context will directly control the returned file. If the context is
// cancelled, then the file is also closed.
func AlbumArt(ctx context.Context, path string) *File {
	// Prioritize searching for external album arts over reading the album art
	// into memory.
	dir := filepath.Dir(path)

	openCtx, cancelOpen := context.WithCancel(ctx)
	defer cancelOpen()

	waitGroup := sync.WaitGroup{}
	waitGroup.Add(len(coverFiles))

	go func() {
		waitGroup.Wait()
		cancelOpen()
	}()

	results := make(chan File)

	for _, coverFile := range coverFiles {
		go func(coverFile string) {
			defer waitGroup.Done()

			f, err := os.Open(filepath.Join(dir, coverFile))
			if err != nil {
				return
			}

			file := File{
				ReadCloser: f,
				Extension:  normalizeExt(filepath.Ext(coverFile)),
			}

			select {
			case results <- file:
				// Stop prematurely.
				cancelOpen()
				// Close the file when the context is done.
				closeWhenDone(ctx, f)
				// done
			case <-openCtx.Done():
				f.Close()
			}
		}(coverFile)
	}

	select {
	case file := <-results:
		return &file
	case <-openCtx.Done():
		// continue
	}

	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	closeWhenDone(ctx, f)

	m, err := tag.ReadFrom(f)
	if err == nil {
		if pic := m.Picture(); pic != nil {
			return &File{
				ReadCloser: ioutil.NopCloser(bytes.NewReader(pic.Data)),
				Extension:  normalizeExt(pic.Ext),
			}
		}
	}

	return nil
}

func closeWhenDone(ctx context.Context, f *os.File) {
	go func() { <-ctx.Done(); f.Close() }()
}

func normalizeExt(ext string) string {
	ext = strings.TrimPrefix(ext, ".")
	ext = strings.ToLower(ext)

	if ext == "jpg" {
		ext = "jpeg"
	}

	return ext
}
