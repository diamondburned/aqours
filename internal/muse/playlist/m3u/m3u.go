package m3u

import (
	"os"
	"path/filepath"
	"time"

	"github.com/diamondburned/aqours/internal/muse/playlist"
	"github.com/ushis/m3u"
)

func init() {
	playlist.RegisterParser(".m3u", Parse)
}

func Parse(path string) (*playlist.Playlist, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	f.SetDeadline(time.Now().Add(15 * time.Second))

	p, err := m3u.Parse(f)
	if err != nil {
		return nil, err
	}

	var pl = playlist.Playlist{
		Name:   basename(path),
		Path:   path,
		Tracks: make([]*playlist.Track, len(p)),
	}

	for i, track := range p {
		var title = track.Title
		if title == "" {
			title = filepath.Base(track.Path)
		}
		if title == "" {
			continue
		}

		pl.Tracks[i] = &playlist.Track{
			Title:    title,
			Length:   time.Duration(track.Time) * time.Second,
			Filepath: track.Path,
		}
	}

	return &pl, nil
}

func basename(path string) string {
	name := filepath.Base(path)
	ext := filepath.Ext(name)
	name = name[:len(name)-len(ext)]
	return name
}
