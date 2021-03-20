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
	OnBitrateChange(bitrate float64)
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
		"--replaygain=album",
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
		Command:    cmd,
		PlayTime:   &TimeContainer{},
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
			s.PlayTime.updatePos(event.Data.(float64))

		case timeRemainingEvent:
			s.PlayTime.updateRem(event.Data.(float64))

		case audioDeviceEvent:
			log.Println("Audio device changed to", event.Data)
		}

		return

	handleAllEvents:
		switch event.Name {
		case "idle":
			// log.Println("Player is idle.")
			// glib.IdleAdd(func() { handler.OnSongFinish() })

		case "end-file":
			// log.Printf(
			// 	"End of file, reason: %q, error: %q %v\n",
			// 	event.Reason, event.Error, event.Data,
			// )
			// Empty reason means not end of file. Don't do anything.
			// Sometimes, when a track ends or we change the track, this
			// event is fired with an empty reason. Thankfully, we could
			// also check for the "idle" event instead, so this event will
			// be used more for errors.
			//
			// For some reason, the stop event behaves a bit erratically.
			if event.Reason != "" && event.Reason != "stop" {
				s.PlayTime.updatePos(0)
				s.PlayTime.updateRem(0)

				glib.IdleAdd(func() {
					s.stopped = true
					handler.OnSongFinish()
				})
			}
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

// TimeContainer wraps an atomic time container.
type TimeContainer struct {
	pos uint64
	rem uint64
}

func (tc *TimeContainer) updatePos(pos float64) {
	atomic.StoreUint64(&tc.pos, math.Float64bits(pos))
}

func (tc *TimeContainer) updateRem(rem float64) {
	atomic.StoreUint64(&tc.rem, math.Float64bits(rem))
}

// Load reads the timestamp atomically.
func (tc *TimeContainer) Load() (pos, rem float64) {
	pos = math.Float64frombits(atomic.LoadUint64(&tc.pos))
	rem = math.Float64frombits(atomic.LoadUint64(&tc.rem))
	return
}
