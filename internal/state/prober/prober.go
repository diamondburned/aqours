package prober

import (
	"log"
	"runtime"

	"github.com/diamondburned/aqours/internal/muse/playlist"
	"github.com/diamondburned/aqours/internal/state"
	"github.com/gotk3/gotk3/glib"
)

// Job is an internal type that allows a track to be copied in a thread-safe way
// for probing.
type Job struct {
	done func() // called in glib
	ptr  *state.Track
	cpy  playlist.Track
}

// NewJob creates a new job. done will be called in the glib main thread.
func NewJob(track *state.Track, done func()) Job {
	return Job{
		ptr:  track,
		done: done,
		cpy:  track.Metadata(),
	}
}

var (
	maxJobs    = runtime.GOMAXPROCS(-1) * 2
	probeQueue = make(chan Job, maxJobs+1)
)

func init() {
	// Go is probably efficient enough to make this a minor issue.
	for i := 0; i < maxJobs; i++ {
		go func() {
			for job := range probeQueue {
				job := job // copy must

				// Probe and update the copy.
				if err := job.cpy.ForceProbe(); err != nil {
					log.Printf("Failed to probe %q: %v", job.cpy.Filepath, err)
					continue
				}

				glib.IdleAdd(func() {
					// Update the original track with the copy.
					job.ptr.UpdateMetadata(job.cpy)
					job.done()
				})
			}
		}()
	}
}

// Queue queues multiple probeJobs. It is thread-safe and non-blocking.
func Queue(jobs ...Job) {
	if len(jobs) == 0 {
		return
	}

	// Try and delegate as many jobs into the queue as possible without spawning
	// goroutines.
	var tried int
TryQueue:
	for _, job := range jobs {
		select {
		case probeQueue <- job:
			tried++
		default:
			break TryQueue
		}
	}

	// Last resort to queue the remaining jobs in goroutines.
	if tried < len(jobs) {
		go func() {
			for _, job := range jobs[tried:] {
				probeQueue <- job
			}
		}()
	}
}
