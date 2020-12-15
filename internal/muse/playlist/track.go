package playlist

import (
	"path/filepath"
	"strconv"
	"time"

	"github.com/diamondburned/aqours/internal/muse/metadata/ffprobe"
)

type Track struct {
	Title  string
	Artist string
	Album  string

	Filepath string `json:",omitempty"`

	Number  int
	Length  time.Duration
	Bitrate int
}

// IsProbed returns true if the track is probed.
func (t Track) IsProbed() bool {
	// We'd maybe want to try and probe all the time.
	return true &&
		(t.Bitrate > 0 && t.Length > 0) &&
		(t.Title != "" && t.Artist != "" && t.Album != "")
}

func (t *Track) Probe() error {
	if t.IsProbed() {
		return nil
	}

	return t.ForceProbe()
}

func (t *Track) ForceProbe() error {
	p, err := ffprobe.Probe(t.Filepath)
	if err != nil {
		// We can still reset the title and try to guess it. We might want to do
		// this if the playlist file has invalid titles.
		t.Title = TitleFromPath(t.Filepath)
		return err
	}

	title := p.TagValue("title")

	// Try and keep the old metadata the same, as playlist loaders might somehow
	// derive it.
	if title == "" {
		return nil
	}

	t.Title = title
	t.Artist = p.TagValue("artist")
	t.Album = p.TagValue("album")
	t.Number = p.TagValueInt("track", t.Number)
	t.Bitrate = p.Format.BitRate
	t.Length = time.Duration(p.Format.Duration * float64(time.Second))

	return nil
}

// TitleFromPath grabs the file basename from the given path, which could be
// used as a title placeholder.
func TitleFromPath(path string) string {
	return trimExt(filepath.Base(path))
}

func trimExt(name string) string {
	ext := filepath.Ext(name)
	return name[:len(name)-len(ext)]
}

func stringOr(str, or string) string {
	if str != "" {
		return str
	}
	return or
}

func intOr(str string, or int) int {
	if n, err := strconv.Atoi(str); err == nil {
		return n
	}
	return or
}
