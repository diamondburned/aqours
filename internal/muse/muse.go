package muse

import (
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
	mpvRead  *mpvReader

	handler EventHandler

	socketPath string
	imagePath  string

	eventChannel chan *mpvipc.Event
	stopEvent    chan struct{}
	playlistPath string
}

func NewSession() (*Session, error) {
	return newMpv()
}

func (s *Session) PlayTrackIndex(n int) error {
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

func (s *Session) SetPlay(playing bool) error {
	return s.Playback.Set("pause", !playing)
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
