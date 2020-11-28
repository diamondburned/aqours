package mpris

import (
	"fmt"
	"log"
	"math"
	"time"

	"github.com/diamondburned/aqours/internal/muse"
	"github.com/diamondburned/aqours/internal/state"
	"github.com/diamondburned/aqours/internal/ui"
	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/prop"
	"github.com/gotk3/gotk3/glib"
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

func trackID(playlist *state.Playlist, trackIx int) dbus.ObjectPath {
	if playlist == nil || trackIx < 0 {
		return dbus.ObjectPath("/org/mpris/MediaPlayer2/TrackList/NoTrack")
	}
	const trackIDfmt = tracksPath + "/%s/%d"
	return dbus.ObjectPath(fmt.Sprintf(trackIDfmt, playlist.Name, trackIx))
}

type player struct {
	*ui.MainWindow
	propQ chan propChange

	// state
	trackID  dbus.ObjectPath
	position microsecond
}

var _ muse.EventHandler = (*player)(nil)

type propChange struct {
	n string
	v interface{}
}

func newPlayer(prop *prop.Properties) *player {
	propQ := make(chan propChange, 10)

	go func() {
		for send := range propQ {
			if err := prop.Set(playerID, send.n, dbus.MakeVariant(send.v)); err != nil {
				log.Println("MRPIS set prop failed:", err)
			}
		}
	}()

	return &player{
		MainWindow: nil,
		propQ:      propQ,
	}
}

// Destroy stops background workers.
func (p *player) Destroy() {
	close(p.propQ)
}

// sendProp asynchronously queues the prop to be sent through DBus. It pops off
// the first item of the queue if it's full.
func (p *player) sendProp(n string, v interface{}) {
	prop := propChange{n, v}

	select {
	case p.propQ <- prop:
		// done
	default:
		// Pop the earliest prop out and replace it with our latest prop.
		<-p.propQ
		p.propQ <- prop

		log.Println("Warning: prop send buffer overflow.")
	}
}

// Muse event handler methods.

var noTrackMetadata = map[string]interface{}{
	"mpris:trackid": trackID(nil, -1),
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

// Volume and Bitrate omitted (inherited).

func (p *player) OnSongFinish(err error) {
	p.MainWindow.OnSongFinish(err)
}

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
	playlist := state.PlayingPlaylist()

	p.trackID = trackID(playlist, i)

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

func (p *player) OnPositionChange(pos float64, total float64) {
	p.MainWindow.OnPositionChange(pos, total)
	p.position = secondsToMicroseconds(pos)
}

// DBus methods.

func (p *player) Next() *dbus.Error {
	glib.IdleAdd(func() { p.Bar.Controls.Buttons.Next.Clicked() })
	return nil
}

func (p *player) Previous() *dbus.Error {
	glib.IdleAdd(func() { p.Bar.Controls.Buttons.Prev.Clicked() })
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
	glib.IdleAdd(func() { p.Bar.Controls.Buttons.Play.Clicked() })
	return nil
}

func (p *player) Seek(us microsecond) *dbus.Error {
	pos := microsecondsToSeconds(p.position) + microsecondsToSeconds(us)
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
