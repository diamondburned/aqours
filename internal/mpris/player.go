package mpris

import (
	"fmt"
	"log"
	"math"
	"time"

	"github.com/diamondburned/aqours/internal/muse"
	"github.com/diamondburned/aqours/internal/state"
	"github.com/diamondburned/aqours/internal/ui"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/prop"
)

type microsecond = int

func secondsToMicroseconds(secs float64) microsecond {
	const us = float64(time.Second / time.Microsecond)
	return int(math.Round(secs * us))
}

func microsecondsToSeconds(usec microsecond) float64 {
	const us = float64(time.Second / time.Microsecond)
	return float64(usec) / us
}

func trackID(trackIx int) dbus.ObjectPath {
	if trackIx < 0 {
		return dbus.ObjectPath("/org/mpris/MediaPlayer2/TrackList/NoTrack")
	}
	const trackIDfmt = tracksPath + "/%d"
	return dbus.ObjectPath(fmt.Sprintf(trackIDfmt, trackIx))
}

type player struct {
	*ui.MainWindow
	propQ chan propChange
	stop  chan struct{}

	// state
	trackID dbus.ObjectPath
	// position microsecond
}

var _ muse.EventHandler = (*player)(nil)

type propChange struct {
	n string
	v interface{}
}

func newPlayer(prop *prop.Properties) *player {
	propQ := make(chan propChange, 10)
	stop := make(chan struct{})

	go func() {
		for {
			select {
			case <-stop:
				return
			case send := <-propQ:
				if err := prop.Set(playerID, send.n, dbus.MakeVariant(send.v)); err != nil {
					log.Println("MRPIS set prop failed:", err)
				}
			}
		}
	}()

	return &player{
		MainWindow: nil,
		propQ:      propQ,
		stop:       stop,
	}
}

// Destroy stops background workers.
func (p *player) Destroy() {
	close(p.stop)
}

// sendProp queues the prop to be sent through DBus. It pops off the first item
// of the queue if it's full.
func (p *player) sendProp(n string, v interface{}) {
	prop := propChange{n, v}

	for {
		select {
		case <-p.stop:
			return
		case p.propQ <- prop:
			return
		default:
			log.Println("Warning: prop send buffer overflow.")

			// Try and pop the earliest prop out.
			select {
			case <-p.propQ:
			default:
			}
		}
	}
}

// Muse event handler methods.

var noTrackMetadata = map[string]interface{}{
	"mpris:trackid": trackID(-1),
}

// I hate implementing this, and I hate dbus. I hate its design. Why the fuck
// would it create a feedback loop when it's waiting for a fucking reply? Why is
// it designed this weirdly? Why can't it just be a stateless asynchronous event
// receiver and state getter? Why does it need to fucking echo shit back? Why
// does it send a few events at the start? Why does my volume all of a sudden
// get kicked to 0.9 before going back to 1? Why is it constantly flipping
// shuffle mode?

func (p *player) SetRepeat(mode state.RepeatMode) {}

func (p *player) SetShuffle(shuffle bool) {}

func (p *player) SetVolume(volume float64) {}

func (p *player) SetMute(mute bool) {}

// Volume, Bitrate and OnSongFinish omitted (inherited).

func (p *player) OnPauseUpdate(pause bool) {
	p.MainWindow.OnPauseUpdate(pause)

	if pause {
		p.sendProp("PlaybackStatus", "Paused")
	} else {
		p.sendProp("PlaybackStatus", "Playing")
	}
}

func (p *player) sendPlaying(state *state.State) {
	i, track := state.NowPlaying()
	p.trackID = trackID(i)

	// If we don't have anything playing...
	if track == nil {
		p.sendProp("Metadata", noTrackMetadata)
		p.sendProp("PlaybackStatus", "Paused")
	} else {
		metadata := track.Metadata()
		p.sendProp("PlaybackStatus", "Playing")
		p.sendProp("Metadata", map[string]interface{}{
			"mpris:trackid":     p.trackID,
			"mpris:length":      metadata.Length.Microseconds(),
			"xesam:title":       metadata.Title,
			"xesam:album":       metadata.Album,
			"xesam:artist":      metadata.Artist,
			"xesam:trackNumber": metadata.Number,
		})
	}
}

// DBus methods.

func (p *player) Next() *dbus.Error {
	glib.IdleAdd(func() { p.Bar.Controls.Buttons.Next.Activate() })
	return nil
}

func (p *player) Previous() *dbus.Error {
	glib.IdleAdd(func() { p.Bar.Controls.Buttons.Prev.Activate() })
	return nil
}

func (p *player) Pause() *dbus.Error {
	glib.IdleAdd(func() { p.Bar.Controls.Buttons.Play.SetPlaying(false) })
	return nil
}

func (p *player) Play() *dbus.Error {
	glib.IdleAdd(func() { p.Bar.Controls.Buttons.Play.SetPlaying(true) })
	return nil
}

func (s *player) Stop() *dbus.Error {
	return errUnimplemented
}

func (p *player) PlayPause() *dbus.Error {
	glib.IdleAdd(func() { p.Bar.Controls.Buttons.Play.Activate() })
	return nil
}

func (p *player) Seek(us microsecond) *dbus.Error {
	s := p.PlaySession()

	pos, _ := s.PlayState.PlayTime()
	pos += microsecondsToSeconds(us)

	glib.IdleAdd(func() { p.MainWindow.Seek(pos) })
	return nil
}

func (p *player) SetPosition(id dbus.ObjectPath, us microsecond) *dbus.Error {
	glib.IdleAdd(func() {
		// Seek if our trackID is not stale.
		if p.trackID == id {
			p.MainWindow.Seek(microsecondsToSeconds(us))
			return
		}
	})

	return nil
}
