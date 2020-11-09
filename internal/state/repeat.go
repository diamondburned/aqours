package state

type RepeatMode uint8

const (
	RepeatNone RepeatMode = iota
	RepeatAll
	RepeatSingle
	repeatLen
)

func enableRepeat(playlist bool) RepeatMode {
	if playlist {
		return RepeatAll
	}
	return RepeatSingle
}

// Cycle returns the next mode to be activated when the repeat button is
// constantly pressed.
func (m RepeatMode) Cycle() RepeatMode {
	return (m + 1) % repeatLen
}
