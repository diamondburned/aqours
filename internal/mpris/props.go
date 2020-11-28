package mpris

import (
	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/prop"
	"github.com/pkg/errors"
)

var playerProps = map[string]*prop.Prop{
	"PlaybackStatus": newWritableProp("Paused", nil),
	"LoopStatus":     newWritableProp("None", unimplementedChangeFn),
	"Rate":           newWritableProp(1.0, unimplementedChangeFn),
	"Shuffle":        newWritableProp(false, unimplementedChangeFn),
	"Metadata":       newWritableProp(noTrackMetadata, nil),
	"Volume":         newWritableProp(1.0, unimplementedChangeFn),
	"Position":       newWritableUnemittedProp(int64(0), nil),
	"MinimumRate":    newWritableProp(1.0, nil),
	"MaximumRate":    newWritableProp(1.0, nil),
	"CanGoNext":      newWritableProp(true, nil),
	"CanGoPrevious":  newWritableProp(true, nil),
	"CanPlay":        newWritableProp(true, nil),
	"CanPause":       newWritableProp(true, nil),
	"CanSeek":        newWritableProp(false, nil),
	"CanControl":     newWritableProp(false, nil),
}

var errUnimplemented = dbus.MakeFailedError(errors.New("unimplemented"))

func unimplementedChangeFn(*prop.Change) *dbus.Error {
	return errUnimplemented
}

func newWritableProp(v interface{}, fn func(*prop.Change) *dbus.Error) *prop.Prop {
	return &prop.Prop{
		Value:    v,
		Writable: true,
		Emit:     prop.EmitTrue,
		Callback: fn,
	}
}

func newWritableUnemittedProp(v interface{}, fn func(*prop.Change) *dbus.Error) *prop.Prop {
	return &prop.Prop{
		Value:    v,
		Writable: true,
		Emit:     prop.EmitFalse,
		Callback: fn,
	}
}
