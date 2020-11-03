package muse

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"time"

	"github.com/DexterLB/mpvipc"
	"github.com/gotk3/gotk3/glib"
	"github.com/pkg/errors"
)

const (
	allEvent uint = iota
	pathEvent
	pauseEvent
	bitrateEvent
)

// EventHandler methods are all called in the glib main thread.
type EventHandler interface {
	OnMPVEvent(event *mpvipc.Event)
	OnPathUpdate(path string)
	OnPauseUpdate(pause bool)
	OnBitrateChange(bitrate int)
}

var tmpdir = filepath.Join(os.TempDir(), "aqours")

// generateUniqueBits generates a small string of unique-ish characters.
func generateUniqueBits(prefix string) string {
	randomBits := make([]byte, 2)
	rand.Read(randomBits)

	nanoBits := make([]byte, 4)
	nano := uint32(time.Now().Unix())
	binary.LittleEndian.PutUint32(nanoBits, nano)

	nanob64 := base64.RawURLEncoding.EncodeToString(nanoBits)
	randb64 := base64.RawURLEncoding.EncodeToString(randomBits)
	return prefix + nanob64 + randb64
}

func generateImageOutDir() string {
	return filepath.Join(tmpdir, "mpv", generateUniqueBits("image-"))
}

func generateMpvSock() string {
	return filepath.Join(tmpdir, "mpv", generateUniqueBits("socket-")+".sock")
}

func newMpv() (*Session, error) {
	sockPath := generateMpvSock()
	imagePath := generateImageOutDir()

	if err := os.MkdirAll(filepath.Dir(sockPath), os.ModePerm); err != nil {
		return nil, errors.Wrap(err, "failed to make socket directory")
	}

	if err := os.MkdirAll(filepath.Dir(imagePath), os.ModePerm); err != nil {
		return nil, errors.Wrap(err, "failed to make image directory")
	}

	cmd := exec.Command(
		"mpv",
		"--idle",
		"--quiet",
		"--pause",
		"--no-input-terminal",
		"--input-ipc-server="+sockPath,
		"--vo=image",
		"--vo-image-format=png",
		"--vo-image-png-compression=9",
		"--vo-image-outdir="+imagePath,
	)

	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	conn := mpvipc.NewConnection(sockPath)

	return &Session{
		Playback:     conn,
		Command:      cmd,
		socketPath:   sockPath,
		imagePath:    imagePath,
		eventChannel: make(chan *mpvipc.Event, 8),
		stopEvent:    make(chan struct{}, 1),
	}, nil
}

func (s *Session) observeProperties(properties map[uint]string) error {
	for id, event := range properties {
		_, err := s.Playback.Call("observe_property_string", id, event)
		if err != nil {
			return errors.Wrapf(err, "failed to observe event %q", event)
		}
	}
	return nil
}

func (s *Session) SetHandler(h EventHandler) {
	s.handler = h
}

func (s *Session) Start() error {
	if err := s.Command.Start(); err != nil {
		return errors.Wrap(err, "failed to start mpv")
	}

	// Spin until the socket exists.
	for {
		_, err := os.Stat(s.socketPath)
		if err == nil {
			break
		}

		runtime.Gosched()
	}

	if err := s.Playback.Open(); err != nil {
		return errors.Wrap(err, "failed to open connection")
	}

	go s.Playback.ListenForEvents(s.eventChannel, s.stopEvent)

	err := s.observeProperties(map[uint]string{
		pathEvent:    "path",
		pauseEvent:   "pause",
		bitrateEvent: "audio-bitrate",
	})
	if err != nil {
		return errors.Wrap(err, "failed to observe properties")
	}

	go func() {
		for event := range s.eventChannel {
			event := event // copy

			if event.Data == nil {
				continue
			}

			switch event.ID {
			case allEvent:
				s.handler.OnMPVEvent(event)
			case pathEvent:
				glib.IdleAdd(func() { s.handler.OnPathUpdate(event.Data.(string)) })

			case pauseEvent:
				log.Println("paused:", event.Data.(string))
				b := parseYesNo(event.Data.(string))
				glib.IdleAdd(func() { s.handler.OnPauseUpdate(b) })

			case bitrateEvent:
				i, _ := strconv.Atoi(event.Data.(string))
				glib.IdleAdd(func() { s.handler.OnBitrateChange(i) })
			}
		}
	}()

	return nil
}

func (s *Session) Stop() {
	s.Command.Process.Signal(os.Interrupt)

	s.Playback.Close()
	s.stopEvent <- struct{}{}

	s.Command.Wait()

	os.Remove(s.socketPath)
	os.RemoveAll(s.imagePath)
}

func parseYesNo(yesno string) bool {
	if yesno == "yes" {
		return true
	}
	return false
}
