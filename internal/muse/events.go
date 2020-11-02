package muse

import "github.com/YouROK/go-mpv/mpv"

const (
	allEvent uint64 = iota
	replyPath
)

type EventHandler interface {
	OnMPVEvent(*mpv.Event)
	OnPathUpdate(string)
}
