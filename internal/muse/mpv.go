package muse

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"time"

	"github.com/DexterLB/mpvipc"
	"github.com/gotk3/gotk3/glib"
	"github.com/pkg/errors"
)

type mpvEvent uint

const (
	allEvent mpvEvent = iota
	pathEvent
	pauseEvent
	bitrateEvent
	timePositionEvent
	timeRemainingEvent
	repeatFileEvent
	repeatPlaylistEvent
)

var propertyMap = map[mpvEvent]string{
	pathEvent:           "path",
	pauseEvent:          "pause",
	bitrateEvent:        "audio-bitrate",
	timePositionEvent:   "time-pos",
	timeRemainingEvent:  "time-remaining",
	repeatPlaylistEvent: "loop-playlist",
	repeatFileEvent:     "loop-file",
}

type mpvLineEvent uint

var mpvLineMatchers = map[mpvLineEvent]*regexp.Regexp{}

type RepeatMode uint8

const (
	RepeatNone RepeatMode = iota
	RepeatAll
	RepeatSingle
	repeatLen
)

func enableRepeat(playlist bool) RepeatMode {
	if playlist {
		return RepeatAll
	}
	return RepeatSingle
}

// Cycle returns the next mode to be activated when the repeat button is
// constantly pressed.
func (m RepeatMode) Cycle() RepeatMode {
	return (m + 1) % repeatLen
}

// EventHandler methods are all called in the glib main thread.
type EventHandler interface {
	OnPathUpdate(playlistPath, songPath string)
	OnPauseUpdate(pause bool)
	OnRepeatChange(repeat RepeatMode)
	OnBitrateChange(bitrate float64)
	OnPositionChange(pos, total float64)
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
		"--no-video",
		// mpv's vo/image backend is a disappointment.
	)

	mpvReader := newMpvReader(os.Stderr, mpvLineMatchers)
	cmd.Stdout = mpvReader
	cmd.Stderr = os.Stderr

	conn := mpvipc.NewConnection(sockPath)

	if err := cmd.Start(); err != nil {
		return nil, errors.Wrap(err, "failed to start mpv")
	}

	// Spin until the socket exists.
	for {
		_, err := os.Stat(sockPath)
		if err == nil {
			break
		}

		runtime.Gosched()
	}

	if err := conn.Open(); err != nil {
		return nil, errors.Wrap(err, "failed to open connection")
	}

	for id, event := range propertyMap {
		_, err := conn.Call("observe_property", id, event)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to observe event %q", event)
		}
	}

	return &Session{
		Playback:     conn,
		Command:      cmd,
		mpvRead:      mpvReader,
		socketPath:   sockPath,
		imagePath:    imagePath,
		eventChannel: make(chan *mpvipc.Event, 8),
		stopEvent:    make(chan struct{}, 1),
	}, nil
}

func (s *Session) SetHandler(h EventHandler) {
	s.handler = h
}

// Start starts all the event listeners in background goroutines. As such, it is
// non-blocking.
func (s *Session) Start() {
	// Copy the handler so the caller cannot change it.
	var handler = s.handler

	// This isn't needed for now.
	// s.mpvRead.Start(func(name mpvLineEvent, matches []string) {})

	go s.Playback.ListenForEvents(s.eventChannel, s.stopEvent)

	go func() {
		// This is kind of racy, but that's about as good as "event-based" as we
		// can get.
		var timeRemaining, timePosition float64

		for event := range s.eventChannel {
			if event.Data == nil {
				continue
			}

			switch mpvEvent(event.ID) {
			case allEvent:
				// noop.

			case pathEvent:
				path := event.Data.(string)
				glib.IdleAdd(func() { handler.OnPathUpdate(s.playlistPath, path) })

			case pauseEvent:
				b := event.Data.(bool)
				glib.IdleAdd(func() { handler.OnPauseUpdate(b) })

			case bitrateEvent:
				i := event.Data.(float64)
				glib.IdleAdd(func() { handler.OnBitrateChange(i) })

			case timePositionEvent:
				timePosition = event.Data.(float64)
				position, total := timePosition, timePosition+timeRemaining
				glib.IdleAdd(func() { handler.OnPositionChange(position, total) })

			case timeRemainingEvent:
				timeRemaining = event.Data.(float64)
				position, total := timePosition, timePosition+timeRemaining
				glib.IdleAdd(func() { handler.OnPositionChange(position, total) })

			case repeatFileEvent:
				sf := s.validRepeatValue(false, event.Data)
				glib.IdleAdd(func() { handler.OnRepeatChange(sf) })

			case repeatPlaylistEvent:
				sf := s.validRepeatValue(true, event.Data)
				glib.IdleAdd(func() { handler.OnRepeatChange(sf) })
			}
		}
	}()
}

func (s *Session) validRepeatValue(pl bool, v interface{}) (repeat RepeatMode) {
	log.Println("repeat value:", v)

	repeat = RepeatNone

	switch v := v.(type) {
	case bool:
		if v {
			repeat = enableRepeat(pl)
		}
	case string:
		switch v {
		case "inf", "force":
			repeat = enableRepeat(pl)
		}
	}

	if b, ok := v.(bool); ok && b {
	}

	// This makes no guarantees that mpv's state is behaving as expected,
	// because it's a lot of code.

	return
}

func (s *Session) Stop() {
	s.Command.Process.Signal(os.Interrupt)
	s.mpvRead.Close()

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
