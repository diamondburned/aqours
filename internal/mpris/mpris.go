package mpris

import (
	"github.com/diamondburned/aqours/internal/muse"
	"github.com/diamondburned/aqours/internal/state"
	"github.com/diamondburned/aqours/internal/ui"
	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"
	"github.com/godbus/dbus/v5/prop"
	"github.com/pkg/errors"
)

const (
	aqoursPath = "/com/github/diamondburned/aqours"
	tracksPath = aqoursPath + "/Tracks"

	mprisPath = "/org/mpris/MediaPlayer2"

	introspectID = "org.freedesktop.DBus.Introspectable"
	mprisID      = "org.mpris.MediaPlayer2"
	playerID     = mprisID + ".Player"
	aqoursID     = mprisID + ".aqours"
)

// Conn is a single MPRIS DBus connection.
type Conn struct {
	conn   *dbus.Conn
	player *player
}

// New creates a new MPRIS connection. It is not ready to be used until
// PassthroughEvents is called.
func New() (*Conn, error) {
	c, err := newConn()
	if err == nil {
		return c, nil
	}

	c.Close()
	return nil, err
}

func newConn() (*Conn, error) {
	s, err := dbus.SessionBus()
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect to session bus")
	}

	props := map[string]map[string]*prop.Prop{
		playerID: playerProps,
	}

	p, err := prop.Export(s, mprisPath, props)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create DBus properties")
	}

	conn := Conn{
		conn:   s,
		player: newPlayer(p),
	}

	if err := s.Export(conn.player, mprisPath, playerID); err != nil {
		return &conn, errors.Wrap(err, "failed to export the MPRIS Player")
	}

	if err := s.Export(introspectionXML, mprisPath, introspectID); err != nil {
		return &conn, errors.Wrap(err, "failed to export introspection.xml")
	}

	reply, err := s.RequestName(aqoursID, dbus.NameFlagDoNotQueue)
	if err != nil {
		return &conn, errors.Wrap(err, "failed to request name")
	}

	if reply != dbus.RequestNameReplyPrimaryOwner {
		return &conn, errors.New("requested name is not primary, name already taken")
	}

	return &conn, nil
}

// Close closes the current DBus connection and destroys background workers. If
// c is nil, then Close returns nil.
func (c *Conn) Close() error {
	if c == nil {
		return nil
	}

	c.player.Destroy()
	return c.conn.Close()
}

// Update signals to MPRIS to update the properties from state.
func (c *Conn) Update(s *state.State) {
	c.player.sendPlaying(s)
}

// PassthroughEvents passes-through events from the returned EventHandler into
// MainWindow. Events that are intercepted will update the MPRIS state after
// they're updated in the UI using w's methods.
func (c *Conn) PassthroughEvents(w *ui.MainWindow) muse.EventHandler {
	c.player.MainWindow = w
	return c.player
}

const introspectionXML introspect.Introspectable = `
<node>
	<interface name="org.mpris.MediaPlayer2.Player">
		<method name="Next">
		</method>
		<method name="Previous">
		</method>
		<method name="Pause">
		</method>
		<method name="PlayPause">
		</method>
		<method name="Stop">
		</method>
		<method name="Play">
		</method>
		<method name="Seek">
			<arg type="x" name="Offset" direction="in"/>
		</method>
		<method name="SetPosition">
			<arg type="o" name="TrackId" direction="in"/>
			<arg type="x" name="Offset" direction="in"/>
		</method>
		<method name="OpenUri">
			<arg type="s" name="Uri" direction="in"/>
		</method>
		<signal name="Seeked">
			<arg type="x" name="Position" direction="out"/>
		</signal>
		<property name="PlaybackStatus" type="s" access="read"/>
		<property name="LoopStatus" type="s" access="readwrite"/>
		<property name="Rate" type="d" access="readwrite"/>
		<property name="Shuffle" type="b" access="readwrite"/>
		<property name="Metadata" type="a{sv}" access="read"/>
		<property name="Volume" type="d" access="readwrite"/>
		<property name="Position" type="x" access="read"/>
		<property name="MinimumRate" type="d" access="read"/>
		<property name="MaximumRate" type="d" access="read"/>
		<property name="CanGoNext" type="b" access="read"/>
		<property name="CanGoPrevious" type="b" access="read"/>
		<property name="CanPlay" type="b" access="read"/>
		<property name="CanPause" type="b" access="read"/>
		<property name="CanSeek" type="b" access="read"/>
		<property name="CanControl" type="b" access="read"/>
	</interface>
</node>
`
