package main

import (
	"log"
	"os"

	"github.com/diamondburned/aqours/internal/muse"
	"github.com/diamondburned/aqours/internal/muse/metadata/ffmpeg"
	"github.com/diamondburned/aqours/internal/muse/playlist"
	"github.com/diamondburned/aqours/internal/state"
	"github.com/diamondburned/aqours/internal/ui"
	"github.com/diamondburned/handy"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
)

func main() {
	ses, err := muse.NewSession()
	if err != nil {
		log.Fatalln("Failed to create mpv session:", err)
	}

	app, err := gtk.ApplicationNew("com.github.diamondburned.aqours", 0)
	if err != nil {
		log.Fatalln("Failed to create a GtkApplication:", err)
	}

	st, err := state.ReadFromFile()
	if err != nil {
		st = state.NewState()
		log.Printf("failed to restore state (%v); creating a new one.\n", err)
	}

	app.Connect("activate", func() {
		handy.Init()

		w, err := ui.NewMainWindow(app, ses)
		if err != nil {
			log.Fatalln("Failed to create main window:", err)
		}

		w.UseState(st)

		// Start is non-blocking, as it should be when ran inside the main
		// thread.
		ses.Start()

		w.Show()
		app.AddWindow(w)

		// Try to save the state and all playlists every 30 seconds.
		glib.TimeoutAdd(30*1000, func() bool {
			st.Save()
			savePlaylists(st.Playlists())
			return true
		})

		w.Connect("destroy", func() {
			ses.Stop()
			ffmpeg.StopAll()
			st.ForceSave()
		})
	})

	if exitCode := app.Run(os.Args); exitCode > 0 {
		os.Exit(exitCode)
	}
}

func savePlaylists(pl []*playlist.Playlist) {
	for _, playlist := range pl {
		playlist.Save(func(err error) {
			log.Printf("failed to periodically save playlist %q: %v\n", playlist.Name, err)
		})
	}
}
