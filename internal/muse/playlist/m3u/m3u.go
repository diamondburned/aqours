package m3u

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/diamondburned/aqours/internal/muse/playlist"
	"github.com/pkg/errors"
	"github.com/ushis/m3u"
)

func init() {
	playlist.Register(".m3u", Parse, Write)
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
		Tracks: make([]playlist.Track, len(p)),
	}

	for i, track := range p {
		var title = track.Title
		if title == "" {
			title = filepath.Base(track.Path)
		}
		if title == "" {
			continue
		}

		pl.Tracks[i] = playlist.Track{
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

	u, err := url.PathUnescape(name)
	if err != nil {
		return name
	}
	return u
}

type ErrNamePathMismatch struct {
	name string
	base string
}

var _ playlist.FixableError = (*ErrNamePathMismatch)(nil)

func (err ErrNamePathMismatch) Error() string {
	return fmt.Sprintf("Name %q mismatches path filename %q", err.name, err.base)
}

func (err ErrNamePathMismatch) Fix(pl *playlist.Playlist) {
	oldPath := pl.Path
	// Clean up the old playlist file.
	go func() { os.Rename(oldPath, makeDotfile(oldPath)) }()

	pl.Path = pathFromName(pl.Path, pl.Name)
}

func makeDotfile(path string) string {
	dir, name := filepath.Split(path)
	return filepath.Join(dir, fmt.Sprintf(".%s.bak", name))
}

var slashesc = strings.NewReplacer("/", "∕", `\`, "⧵").Replace

func pathFromName(path, name string) string {
	dirnm := filepath.Dir(path)
	fname := fmt.Sprintf("%s.m3u", slashesc(name))
	return filepath.Join(dirnm, fname)
}

func Write(p *playlist.Playlist, done func(error)) error {
	// Verify.
	if p.Path != pathFromName(p.Path, p.Name) {
		return ErrNamePathMismatch{
			name: p.Name,
			base: basename(p.Path),
		}
	}

	var plist = make(m3u.Playlist, len(p.Tracks))

	for i, track := range p.Tracks {
		plist[i] = m3u.Track{
			Title: track.Title,
			Path:  track.Filepath,
			Time:  int64(track.Length.Seconds()),
		}
	}

	go func() {
		f, err := os.Create(p.Path)
		if err != nil {
			done(errors.Wrap(err, "failed to create playlist file"))
			return
		}
		defer f.Close()

		buf := bufio.NewWriter(f)

		if _, err := plist.WriteTo(f); err != nil {
			done(errors.Wrap(err, "failed to write playlist"))
			return
		}

		if err := buf.Flush(); err != nil {
			done(errors.Wrap(err, "failed to flush"))
			return
		}

		done(nil)
	}()

	return nil
}
