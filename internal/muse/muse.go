package muse

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/YouROK/go-mpv/mpv"
	"github.com/pkg/errors"

	"github.com/diamondburned/aqours/internal/muse/cgoutil"
	_ "github.com/diamondburned/aqours/internal/muse/playlist/audpl"
	_ "github.com/diamondburned/aqours/internal/muse/playlist/m3u"
)

var ErrNoPlaylistLoaded = errors.New("no playlist loaded")

var tmpdir = filepath.Join(os.TempDir(), "aqours")

func generateImageOutDir() string {
	var randomBits = make([]byte, 4)
	rand.Read(randomBits)

	lastDir := fmt.Sprintf(
		"image-%d-%s",
		time.Now().UnixNano(),
		base64.URLEncoding.EncodeToString(randomBits),
	)

	return filepath.Join(tmpdir, "mpv", lastDir)
}

type Session struct {
	Playback *mpv.Mpv
	Handler  EventHandler

	listenerClose chan struct{}
	playlistPath  string
}

func NewSession() (*Session, error) {
	playbackMpv, err := newMpv()
	if err != nil {
		return nil, errors.Wrap(err, "failed to make playback mpv")
	}

	s := &Session{
		Playback:      playbackMpv,
		listenerClose: make(chan struct{}, 1),
	}

	if err := s.setProperties(map[string]string{"vid": "no"}); err != nil {
		return nil, err
	}

	err = s.observeProperties(map[uint64]string{
		replyPath: "path",
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to observe properties")
	}

	go func() {
		for {
			ev := s.Playback.WaitEvent(1000)
			if ev == nil || s.Handler == nil {
				select {
				case <-s.listenerClose:
					return
				default:
					continue
				}
			}

			switch ev.Reply_Userdata {
			case allEvent:
				s.Handler.OnMPVEvent(ev)
			case replyPath:
				log.Printf("%q\n", cgoutil.GoStrings(ev.Data))
				// s.Handler.OnPathUpdate(cgoutil.GoString(ev.Data))
			}
		}
	}()

	return s, nil
}

func (s *Session) observeProperties(props map[uint64]string) error {
	for replyID, name := range props {
		if err := s.Playback.ObserveProperty(replyID, name, mpv.FORMAT_STRING); err != nil {
			return errors.Wrapf(err, "failed to observe property %q", name)
		}
	}
	return nil
}

func (s *Session) setProperties(props map[string]string) error {
	for k, v := range props {
		if err := s.Playback.SetPropertyString(k, v); err != nil {
			return errors.Wrapf(err, "failed to set %q=%q", k, v)
		}
	}
	return nil
}

func (s *Session) Shutdown() {
	s.listenerClose <- struct{}{}
	s.Playback.TerminateDestroy()
}

func (s *Session) PlayTrack(path string) error {
	return s.Playback.Command([]string{
		"loadfile", path,
	})
}

func (s *Session) SelectPlaylist(path string) error {
	if s.playlistPath == path {
		return nil
	}

	if err := s.Playback.Command([]string{"loadlist", path}); err != nil {
		return err
	}

	s.playlistPath = path
	return nil
}

func (s *Session) SetPlay(playing bool) error {
	return s.Playback.SetProperty("pause", mpv.FORMAT_FLAG, playing)
}

func (s *Session) Previous() error {
	return s.Playback.Command([]string{
		"playlist-prev", "force",
	})
}

func (s *Session) Next() error {
	return s.Playback.Command([]string{
		"playlist-next", "force",
	})
}
