package muse

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
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
	audioDeviceEvent
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
	audioDeviceEvent:   "audio-device",
}

// EventHandler methods are all called in the glib main thread.
type EventHandler interface {
	OnSongFinish(err error)
	OnPauseUpdate(pause bool)
	OnBitrateChange(bitrate float64)
	OnPositionChange(pos, total float64)
}

var tmpdir = filepath.Join(os.TempDir(), "aqours")

func newMpv() (*Session, error) {
	sockPath := filepath.Join(tmpdir, "mpv", "mpv.sock")

	if err := os.MkdirAll(filepath.Dir(sockPath), os.ModePerm); err != nil {
		return nil, errors.Wrap(err, "failed to make socket directory")
	}

	args := []string{
		"--idle",
		"--quiet",
		"--pause",
		"--no-input-terminal",
		"--gapless-audio=weak",
		"--input-ipc-server=" + sockPath,
		"--volume=100",
		"--no-video",
	}

	// Try and support MPV_MPRIS.
	if scripts := os.Getenv("MPV_SCRIPTS"); scripts != "" {
		for _, script := range strings.Split(scripts, ":") {
			args = append(args, "--script="+script)
		}
	}

	cmd := exec.Command("mpv", args...)
	cmd.Env = os.Environ()
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
			cancel()
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

			case audioDeviceEvent:
				log.Println("Audio device changed to", event.Data)
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
		log.Println("Session already stopped; bailing early.")
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
