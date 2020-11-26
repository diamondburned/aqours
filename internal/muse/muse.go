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

// PlayTrack asynchronously loads and plays a file. An error is not returned
// because mpv doesn't seem to return one regardless.
func (s *Session) PlayTrack(path string) {
	_, err := s.Playback.Call("async", "loadfile", path)
	if err != nil {
		log.Println("async loadfile failed:", err)
		return
	}

	if err := s.SetPlay(true); err != nil {
		log.Println("play failed:", err)
	}
}

func (s *Session) Seek(pos float64) error {
	return s.Playback.Set("time-pos", pos)
}

func (s *Session) SetPlay(playing bool) error {
	return s.Playback.Set("pause", !playing)
}

func (s *Session) SetVolume(perc float64) error {
	return s.Playback.Set("volume", perc)
}

func (s *Session) SetMute(muted bool) error {
	return s.Playback.Set("mute", muted)
}
