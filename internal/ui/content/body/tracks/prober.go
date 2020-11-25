package tracks

// func init() {
// 	// Go is probably efficient enough to make this a minor issue.
// 	for i := 0; i < maxJobs; i++ {
// 		go func() {
// 			for job := range probeQueue {
// 				job := job // copy must

// 				// Probe and update the copy.
// 				if err := job.cpy.Probe(); err != nil {
// 					log.Printf("Failed to probe %q: %v", job.cpy.Filepath, err)
// 					continue
// 				}

// 				glib.IdleAdd(func() {
// 					// Update the original track with the copy.
// 					*job.ptr = job.cpy

// 					// Update the list entry afterwards.
// 					// TODO: check invalidation.
// 					job.row.setListStore(job.ptr, job.list.Store)

// 					// Mark playlist as unsaved.
// 					job.list.Playlist.SetUnsaved()
// 				})
// 			}
// 		}()
// 	}
// }
