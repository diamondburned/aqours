package main

import (
	"log"
	"os"

	"github.com/diamondburned/aqours/internal/muse"
	"github.com/diamondburned/aqours/internal/muse/metadata/ffmpeg"
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
		saveID, _ := glib.TimeoutAdd(30*1000, func() bool {
			st.SaveState()
			w.SaveAllPlaylists()
			return true
		})

		w.Connect("destroy", func() {
			glib.SourceRemove(saveID) // remove callback before retrying
			ses.Stop()
			ffmpeg.StopAll()
			st.SaveState()
			w.SaveAllPlaylists()
		})
	})

	if exitCode := app.Run(os.Args); exitCode > 0 {
		os.Exit(exitCode)
	}
}
