package tracks

import (
	"log"
	"runtime"

	"github.com/diamondburned/aqours/internal/muse/playlist"
	"github.com/gotk3/gotk3/glib"
)

// probeJob is an internal type that allows a track to be copied in a
// thread-safe way for probing.
type probeJob struct {
	list *TrackList
	row  *TrackRow
	ptr  *playlist.Track
	cpy  playlist.Track
}

func newProbeJob(list *TrackList, tr *playlist.Track, r *TrackRow) probeJob {
	return probeJob{
		list: list,
		row:  r,
		ptr:  tr,
		cpy:  *tr,
	}
}

var (
	maxJobs    = runtime.GOMAXPROCS(-1)
	probeQueue = make(chan probeJob, maxJobs+1)
)

func init() {
	// Go is probably efficient enough to make this a minor issue.
	for i := 0; i < maxJobs; i++ {
		go func() {
			for job := range probeQueue {
				job := job // copy must

				// Probe and update the copy.
				if err := job.cpy.Probe(); err != nil {
					log.Printf("Failed to probe %q: %v", job.cpy.Filepath, err)
					continue
				}

				glib.IdleAdd(func() {
					// Update the original track with the copy.
					*job.ptr = job.cpy

					// Update the list entry afterwards.
					// TODO: check invalidation.
					job.row.setListStore(job.ptr, job.list.Store)

					// Mark playlist as unsaved.
					job.list.Playlist.SetUnsaved()
				})
			}
		}()
	}
}

// queueProbeJobs queues multiple probeJobs. It is thread-safe and non-blocking.
func queueProbeJobs(jobs ...probeJob) {
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
