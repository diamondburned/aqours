package muse

import (
	"github.com/YouROK/go-mpv/mpv"
	"github.com/pkg/errors"
)

func newMpv() (*mpv.Mpv, error) {
	mpvSession := mpv.Create()
	if mpvSession == nil {
		return nil, errors.New("failed to create mpv")
	}

	if err := mpvSession.Initialize(); err != nil {
		return nil, errors.Wrap(err, "failed to initialize mpv")
	}

	return mpvSession, nil
}
