package playlist

import (
	"errors"
	"path/filepath"
	"sort"
)

type PlaylistReader func(path string) (*Playlist, error)

var playlistReaders = map[string]PlaylistReader{}

func SupportedExtensions() []string {
	var exts = make([]string, 0, len(playlistReaders))
	for ext := range playlistReaders {
		exts = append(exts, ext)
	}
	sort.Strings(exts)
	return exts
}

func RegisterParser(fileExt string, r PlaylistReader) {
	playlistReaders[fileExt] = r
}

func ParseFile(path string) (*Playlist, error) {
	fn, ok := playlistReaders[filepath.Ext(path)]
	if !ok {
		return nil, errors.New("unknown format")
	}

	return fn(path)
}

type Playlist struct {
	Name   string
	Path   string
	Tracks []*Track
}
