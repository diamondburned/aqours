package playlist

import (
	"os"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/dhowden/tag"
	"github.com/diamondburned/aqours/internal/muse/metadata/ffprobe"
	"github.com/pkg/errors"
)

type Track struct {
	Title   string
	Artist  string
	Album   string
	Number  int
	Length  time.Duration
	Bitrate int

	Filepath string
}

// IsProbed returns true if the track is probed.
func (t *Track) IsProbed() bool {
	return t.Bitrate > 0 && t.Length > 0 && !(t.Title == "" && t.Artist == "" && t.Album == "")
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
		return err
	}

	t.Title = stringOr(p.Format.Tags["title"], t.Title)
	t.Artist = stringOr(p.Format.Tags["artist"], t.Artist)
	t.Album = stringOr(p.Format.Tags["album"], t.Album)
	t.Number = intOr(p.Format.Tags["track"], t.Number)
	t.Bitrate = intOr(p.Format.BitRate, t.Bitrate)

	if secs, err := strconv.ParseFloat(p.Format.Duration, 64); err == nil {
		t.Length = time.Duration(secs * float64(time.Second))
	}

	return nil
}

// AlbumArt queries for an album art and read everything INTO MEMORY! It returns
// nil both values if there is no album art.
func (t *Track) AlbumArt() (*tag.Picture, error) {
	f, err := os.Open(t.Filepath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open file")
	}
	defer f.Close()

	// Use a 1 minute timeout.
	f.SetDeadline(time.Now().Add(time.Minute))

	m, err := tag.ReadFrom(f)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read tag")
	}

	return m.Picture(), nil
}

var maxJobs = runtime.GOMAXPROCS(-1)

// BatchProbe batch probes the given slice of track pointers. Although a slice
// of pointers are given, the Probe method will actually be called on a copy of
// the track. The probed callback should therefore reapply the track.
func BatchProbe(tracks []*Track, probed func(*Track, error)) {
	queue := make(chan *Track, maxJobs)
	waitg := sync.WaitGroup{}
	waitg.Add(maxJobs)

	for i := 0; i < maxJobs; i++ {
		go func() {
			defer waitg.Done()

			for track := range queue {
				copyTrack := *track
				probed(&copyTrack, copyTrack.Probe())
			}
		}()
	}

	for _, track := range tracks {
		if track.IsProbed() {
			continue
		}
		queue <- track
	}

	close(queue)
	waitg.Wait()
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
