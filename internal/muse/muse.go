package muse

import (
	"log"
	"os/exec"

	"github.com/DexterLB/mpvipc"
	"github.com/pkg/errors"

	_ "github.com/diamondburned/aqours/internal/muse/playlist/audpl"
	_ "github.com/diamondburned/aqours/internal/muse/playlist/m3u"
)

var ErrNoPlaylistLoaded = errors.New("no playlist loaded")

type Session struct {
	Playback   *mpvipc.Connection
	Command    *exec.Cmd
	handler    EventHandler
	socketPath string

	// OnAsyncError is called on both nil and non-nil.
	OnAsyncError func(error)

	nextSong string
	stopped  bool
}

func NewSession() (*Session, error) {
	return newMpv()
}

// PlayTrack asynchronously loads and plays a file. An error is not returned
// because mpv doesn't seem to return one regardless.
func (s *Session) PlayTrack(path, next string) {
	// We only need to play if the path to be loaded matches the path that's
	// already next in the playlist. Unless we're not stopped when the song is
	// changed, which possibly means that it's a user-requested action.
	if !s.stopped || s.nextSong != path {
		log.Println("Force loading path.")

		if err := s.loadFile(path, false); err != nil {
			log.Println("async loadfile failed:", err)
			return
		}

		if err := s.SetPlay(true); err != nil {
			log.Println("play failed:", err)
		}
	}

	s.stopped = false

	// Preload the next file.
	s.nextSong = next
	if next != "" {
		if err := s.loadFile(next, true); err != nil {
			log.Println("async loadfile next track failed:", err)
			return
		}
	}
}

func (s *Session) loadFile(file string, toAppend bool) (err error) {
	errFn := func(v interface{}, err error) { s.OnAsyncError(err) }

	if toAppend {
		err = s.Playback.CallAsync(errFn, "async", "loadfile", file, "append")
	} else {
		err = s.Playback.CallAsync(errFn, "async", "loadfile", file)
	}

	return
}

func (s *Session) Seek(pos float64) error {
	return s.Playback.SetAsync("time-pos", pos, s.OnAsyncError)
}

func (s *Session) SetPlay(playing bool) error {
	s.stopped = false
	return s.Playback.SetAsync("pause", !playing, s.OnAsyncError)
}

func (s *Session) SetVolume(perc float64) error {
	return s.Playback.SetAsync("volume", perc, s.OnAsyncError)
}

func (s *Session) SetMute(muted bool) error {
	return s.Playback.SetAsync("mute", muted, s.OnAsyncError)
}
