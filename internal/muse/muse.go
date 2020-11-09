package muse

import (
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

	handler EventHandler

	socketPath   string
	eventChannel chan *mpvipc.Event
	stopEvent    chan struct{}
}

func NewSession() (*Session, error) {
	return newMpv()
}

func (s *Session) PlayTrack(path string) error {
	_, err := s.Playback.Call("loadfile", path)
	if err != nil {
		return err
	}

	return s.SetPlay(true)
}

func (s *Session) Seek(pos float64) error {
	return s.Playback.Set("time-pos", pos)
}

func (s *Session) SetPlay(playing bool) error {
	return s.Playback.Set("pause", !playing)
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
