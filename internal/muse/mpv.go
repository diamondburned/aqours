package muse

import (
	"context"
	"log"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
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
	OnSongFinish()
	OnPauseUpdate(pause bool)
}

var tmpdir = filepath.Join(os.TempDir(), "aqours")

func newMpv() (*Session, error) {
	sockPath := filepath.Join(tmpdir, "mpv", "mpv.sock")

	if err := os.MkdirAll(filepath.Dir(sockPath), os.ModePerm); err != nil {
		return nil, errors.Wrap(err, "failed to make socket directory")
	}

	// Trust Gtk in doing the right thing.
	if err := os.RemoveAll(sockPath); err != nil {
		return nil, errors.Wrap(err, "failed to clean up socket")
	}

	args := []string{
		"--idle",
		"--quiet",
		"--pause",
		"--no-input-terminal",
		"--loop-playlist=no",
		"--gapless-audio=weak",
		"--replaygain=track",
		"--replaygain-clip=no",
		"--ad=lavc:*",
		"--input-ipc-server=" + sockPath,
		"--volume=100",
		"--volume-max=100",
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
		Playback:   conn,
		PlayState:  &PlayState{},
		Command:    cmd,
		socketPath: sockPath,
		OnAsyncError: func(err error) {
			if err != nil {
				log.Println("mpv async error:", err)
			}
		},
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

	s.Playback.ListenForEvents(func(event *mpvipc.Event) {
		if event.Error != "" {
			log.Println("Error in event:", event.Error)
		}

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
			s.PlayState.updateBitrate(event.Data.(float64))

		case timePositionEvent:
			s.PlayState.updatePos(event.Data.(float64))

		case timeRemainingEvent:
			s.PlayState.updateRem(event.Data.(float64))

		case audioDeviceEvent:
			log.Println("Audio device changed to", event.Data)
		}

		return

	handleAllEvents:
		switch event.Name {
		case "idle":
			// log.Println("Player is idle.")
			// glib.IdleAdd(func() { handler.OnSongFinish() })

		case "start-file":
			// For some reason, the end-file event behaves a bit erratically, so
			// we use start-file.
			s.PlayState.updatePos(0)
			s.PlayState.updateRem(0)
			s.PlayState.updateBitrate(0)

			glib.IdleAdd(func() {
				// Edge-case when we force playing; because we invoked this
				// action, we don't trigger the callback.
				if s.forced {
					s.forced = false
					return
				}

				s.stopped = true
				handler.OnSongFinish()
			})
		}
	})
}

// Stop stops the mpv session. It does nothing if it's called more than once. A
// stopped session cannot be reused.
func (s *Session) Stop() {
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

// PlayState wraps the current playback state.
type PlayState struct {
	btr uint64
	pos uint64
	rem uint64
}

func (tc *PlayState) updatePos(pos float64) {
	atomic.StoreUint64(&tc.pos, math.Float64bits(pos))
}

func (tc *PlayState) updateRem(rem float64) {
	atomic.StoreUint64(&tc.rem, math.Float64bits(rem))
}

func (tc *PlayState) updateBitrate(btr float64) {
	atomic.StoreUint64(&tc.btr, math.Float64bits(btr))
}

// Bitrate reads the bitrate atomically.
func (tc *PlayState) Bitrate() float64 {
	return math.Float64frombits(atomic.LoadUint64(&tc.btr))
}

// PlayTime reads the playback timestamps atomically.
func (tc *PlayState) PlayTime() (pos, rem float64) {
	pos = math.Float64frombits(atomic.LoadUint64(&tc.pos))
	rem = math.Float64frombits(atomic.LoadUint64(&tc.rem))
	return
}
