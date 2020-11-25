package main

import (
	"log"
	"os"
	"sync"

	"github.com/diamondburned/aqours/internal/muse"
	"github.com/diamondburned/aqours/internal/state"
	"github.com/diamondburned/aqours/internal/ui"
	"github.com/diamondburned/handy"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
)

const appFlags = glib.APPLICATION_FLAGS_NONE

func main() {
	app, err := gtk.ApplicationNew("com.github.diamondburned.aqours", appFlags)
	if err != nil {
		log.Fatalln("Failed to create a GtkApplication:", err)
	}

	// GtkApplication's single instance API is weird: it uses some DBus IPC
	// fuckery to trigger activate a second time. We could use Go's sync.Once to
	// keep ensuring single-instance.
	//
	// Technically, the usage of sync.Once is overkill here, but who cares, it's
	// cleaner.
	var singleInstance sync.Once

	app.Connect("activate", func() {
		singleInstance.Do(func() { activate(app) })
	})

	if exitCode := app.Run(os.Args); exitCode > 0 {
		os.Exit(exitCode)
	}
}

func activate(app *gtk.Application) {
	handy.Init()

	ses, err := muse.NewSession()
	if err != nil {
		log.Fatalln("Failed to create mpv session:", err)
	}

	st, err := state.ReadFromFile()
	if err != nil {
		log.Printf("failed to restore state (%v); creating a new one.\n", err)
		st = state.NewState()
	}

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
		st.SaveState()
		w.SaveAllPlaylists()
		return true
	})

	w.Connect("destroy", func() {
		ses.Stop()
		st.SaveAll()
	})
}
