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

// ShuffleQueue shuffles the given list of track indices.
func ShuffleQueue(queue []int) {
	rand.Shuffle(len(queue), func(i, j int) {
		queue[i], queue[j] = queue[j], queue[i]
	})
}

// ResetQueue resets the queue to the usual incremental order.
func ResetQueue(queue []int) {
	for i := 0; i < len(queue); i++ {
		queue[i] = i
	}
}
