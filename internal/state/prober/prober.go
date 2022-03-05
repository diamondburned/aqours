package prober

import (
	"log"
	"sync"

	"github.com/diamondburned/aqours/internal/muse/playlist"
	"github.com/diamondburned/aqours/internal/state"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
)

// Job is an internal type that allows a track to be copied in a thread-safe way
// for probing.
type Job struct {
	done func() // called in glib
	ptr  *state.Track
	cpy  playlist.Track
	// Force, if true, forces a reprobe.
	Force bool
}

// NewJob creates a new job. done will be called in the glib main thread.
func NewJob(track *state.Track, done func()) Job {
	return Job{
		ptr:  track,
		done: done,
		cpy:  track.Metadata(),
	}
}

// This is quite arbitrary, but it should be fast enough on a local disk and
// doesn't clog much on a remote mount.
var maxJobs = 4

// Variables needed for dynamically scaling workers.
var (
	runningMut   sync.Mutex
	probeQueue   chan Job
	probingGroup sync.WaitGroup
)

func ensureRunning() {
	// log.Println("worker group++")
	probingGroup.Add(1)

	runningMut.Lock()
	defer runningMut.Unlock()

	if probeQueue != nil {
		return
	}

	probeQueue = make(chan Job)
	startRunning(probeQueue)
}

func stopRunning() {
	// log.Println("worker group--")
	probingGroup.Done()
}

func startRunning(queue <-chan Job) {
	// log.Println("starting prober workers")

	go func() {
		probingGroup.Wait()

		runningMut.Lock()
		defer runningMut.Unlock()

		// Kill all current workers.
		if queue == probeQueue {
			close(probeQueue)
			probeQueue = nil
		}
	}()

	// Go is probably efficient enough to make this a minor issue.
	for i := 0; i < maxJobs; i++ {
		go func() {
			// log.Println("worker started")
			// defer log.Println("worker stopped")

			for job := range queue {
				job := job // copy for IdleAdd

				var err error
				if job.Force {
					// Probe and update the copy.
					err = job.cpy.ForceProbe()
				} else {
					err = job.cpy.Probe()
				}

				if err != nil {
					log.Printf("error probing %q: %v", job.cpy.Filepath, err)
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

	go func() {
		ensureRunning()
		defer stopRunning()

		for _, job := range jobs {
			probeQueue <- job
		}
	}()
}
