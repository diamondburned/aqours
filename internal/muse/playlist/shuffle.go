package playlist

import (
	"encoding/binary"
	"math/rand"
	"time"

	cryptorand "crypto/rand"
)

func init() {
	rand.Seed(trueRandSeed())
}

// Meme.
func trueRandSeed() (seed int64) {
	err := binary.Read(cryptorand.Reader, binary.LittleEndian, &seed)
	if err == nil {
		return
	}
	return time.Now().UnixNano()
}

// TODO: stateful shuffler
