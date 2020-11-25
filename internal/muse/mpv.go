package muse

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"fmt"
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
	pauseEvent
	bitrateEvent
	timePositionEvent
	timeRemainingEvent
)

var events = []string{
	"idle",
	"end-file",
}

var propertyMap = map[mpvEvent]string{
	pauseEvent:         "pause",
	bitrateEvent:       "audio-bitrate",
	timePositionEvent:  "time-pos",
	timeRemainingEvent: "time-remaining",
}

type mpvLineEvent uint

var mpvLineMatchers = map[mpvLineEvent]*regexp.Regexp{}

// EventHandler methods are all called in the glib main thread.
type EventHandler interface {
	OnSongFinish(err error)
	OnPauseUpdate(pause bool)
	OnBitrateChange(bitrate float64)
	OnPositionChange(pos, total float64)
}

var tmpdir = filepath.Join(os.TempDir(), "aqours")

// generateUniqueBits generates a small string of unique-ish characters.
func generateUniqueBits() string {
	randomBits := make([]byte, 2)
	rand.Read(randomBits)

	nanoBits := make([]byte, 4)
	nano := uint32(time.Now().Unix())
	binary.LittleEndian.PutUint32(nanoBits, nano)

	nanob64 := base64.RawURLEncoding.EncodeToString(nanoBits)
	randb64 := base64.RawURLEncoding.EncodeToString(randomBits)
	return nanob64 + randb64
}

func generateMpvSock() string {
	return filepath.Join(tmpdir, "mpv", generateUniqueBits()+".sock")
}

func newMpv() (*Session, error) {
	sockPath := generateMpvSock()

	if err := os.MkdirAll(filepath.Dir(sockPath), os.ModePerm); err != nil {
		return nil, errors.Wrap(err, "failed to make socket directory")
	}

	cmd := exec.Command(
		"mpv",
		"--idle",
		"--quiet",
		"--pause",
		"--no-input-terminal",
		"--gapless-audio=weak",
		"--input-ipc-server="+sockPath,
		"--no-video",
		// mpv's vo/image backend is a disappointment.
	)

	cmd.Stderr = os.Stderr

	conn := mpvipc.NewConnection(sockPath)

	if err := cmd.Start(); err != nil {
		return nil, errors.Wrap(err, "failed to start mpv")
	}

	// Give us a 5-second period timeout.
	ctx, cancel := context.WithTimeout(context.TODO(), 5*time.Second)
	defer cancel()

	// Spin until we can connect.
	var err error
RetryOpen:
	for {
		err = conn.Open()
		if err == nil {
			break RetryOpen
		}
		select {
		case <-ctx.Done():
			break RetryOpen
		default:
			runtime.Gosched()
			continue RetryOpen
		}
	}

	if err != nil {
		return nil, errors.Wrap(err, "failed to open connection")
	}

	log.Println("Connection established at", sockPath)

	for _, event := range events {
		_, err := conn.Call("enable_event", event)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to enable event %q", event)
		}
	}

	for id, property := range propertyMap {
		_, err := conn.Call("observe_property", id, property)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to observe property %q", property)
		}
	}

	l, _ := conn.Get("audio-device")
	fmt.Println("Audio device list:", l)

	return &Session{
		Playback:     conn,
		Command:      cmd,
		socketPath:   sockPath,
		eventChannel: make(chan *mpvipc.Event, 8),
		stopEvent:    make(chan struct{}),
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

	go s.Playback.ListenForEvents(s.eventChannel, s.stopEvent)

	go func() {
		// This is kind of racy, but that's about as good as "event-based" as we
		// can get.
		var timeRemaining, timePosition float64

		for event := range s.eventChannel {
			if event.Data == nil {
				goto handleAllEvents
			}

			switch mpvEvent(event.ID) {
			case allEvent:
				goto handleAllEvents

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
			}

			continue

		handleAllEvents:
			switch event.Name {
			case "idle":
				glib.IdleAdd(func() { handler.OnSongFinish(nil) })

			case "end-file":
				// Empty reason means not end of file. Don't do anything.
				// Sometimes, when a track ends or we change the track, this
				// event is fired with an empty reason. Thankfully, we could
				// also check for the "idle" event instead, so this event will
				// be used more for errors.
				if event.Reason != "" {
					var err error
					if event.Reason != "eof" {
						err = fmt.Errorf("error while playing: %s", event.Reason)
					}
					glib.IdleAdd(func() { handler.OnSongFinish(err) })
				}
			}
		}
	}()
}

// Stop stops the mpv session. It does nothing if it's called more than once. A
// stopped session cannot be reused.
func (s *Session) Stop() {
	select {
	case <-s.stopEvent:
		return
	default:
		close(s.stopEvent)
	}

	s.Playback.Close()

	if err := s.Command.Process.Signal(os.Interrupt); err != nil {
		log.Println("Attempted to send SIGINT failed, error occured:", err)
		log.Println("Killing anyway.")

		if err = s.Command.Process.Kill(); err != nil {
			log.Println("Failed to kill mpv:", err)
		}
	} else {
		// Wait for mpv to finish up.
		s.Command.Wait()
	}

	if err := os.Remove(s.socketPath); err != nil {
		log.Println("Failed to clean up socket:", err)
	}
}
