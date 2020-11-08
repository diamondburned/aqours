package muse

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/DexterLB/mpvipc"
	"github.com/pkg/errors"

	_ "github.com/diamondburned/aqours/internal/muse/playlist/audpl"
	_ "github.com/diamondburned/aqours/internal/muse/playlist/m3u"
)

var ErrNoPlaylistLoaded = errors.New("no playlist loaded")

type Session struct {
	Playback *mpvipc.Connection
	Command  *exec.Cmd
	mpvRead  *mpvReader

	handler EventHandler

	socketPath string
	imagePath  string

	eventChannel chan *mpvipc.Event
	stopEvent    chan struct{}
	playlistPath string
	shuffling    bool // required for playlist play index
}

func NewSession() (*Session, error) {
	return newMpv()
}

func (s *Session) PlayTrackIndex(n int) error {
	// Before setting the playlist position, we must unshuffle everything. This
	// has the side effect of reshuffling the playlist after done, but since
	// shuffling is random anyway, it doesn't matter.
	if s.shuffling {
		s.SetShuffle(false)
		defer s.SetShuffle(true)
	}

	if err := s.Playback.Set("playlist-pos", n); err != nil {
		return err
	}

	return s.SetPlay(true)
}

func (s *Session) SelectPlaylist(path string) error {
	if s.playlistPath == path {
		return nil
	}

	if _, err := s.Playback.Call("loadlist", path); err != nil {
		return err
	}

	s.playlistPath = path
	return nil
}

func (s *Session) Previous() error {
	_, err := s.Playback.Call("playlist-prev", "force")
	return err
}

func (s *Session) Next() error {
	_, err := s.Playback.Call("playlist-next", "force")
	return err
}

func (s *Session) Seek(pos float64) error {
	return s.Playback.Set("time-pos", pos)
}

func (s *Session) SetPlay(playing bool) error {
	return s.Playback.Set("pause", !playing)
}

func (s *Session) SetShuffle(shuffle bool) (err error) {
	s.shuffling = shuffle

	if shuffle {
		_, err = s.Playback.Call("playlist-shuffle")
	} else {
		_, err = s.Playback.Call("playlist-unshuffle")
	}

	return err
}

func (s *Session) SetRepeat(repeat RepeatMode) error {
	switch repeat {
	case RepeatNone:
		return makeBatchErrors(
			s.Playback.Set("loop-playlist", "no"),
			s.Playback.Set("loop-file", "no"),
		)
	case RepeatSingle:
		return makeBatchErrors(
			s.Playback.Set("loop-playlist", "no"),
			s.Playback.Set("loop-file", "inf"),
		)
	case RepeatAll:
		return makeBatchErrors(
			s.Playback.Set("loop-playlist", "inf"),
			s.Playback.Set("loop-file", "no"),
		)
	}

	return fmt.Errorf("unknown repeat mode %v", repeat)
}

type batchErrors []error

func makeBatchErrors(errs ...error) error {
	var nonNils = errs[:0]
	for _, err := range errs {
		if err != nil {
			nonNils = append(nonNils, err)
		}
	}

	if len(nonNils) == 0 {
		return nil
	}

	return batchErrors(nonNils)
}

func (b batchErrors) Error() string {
	var errors = make([]string, len(b))
	for i, err := range b {
		errors[i] = err.Error()
	}

	// English moment.
	return strings.Join(errors, ", and ")
}
