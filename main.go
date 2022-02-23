package main

import (
	"fmt"
	"log"
	"os"

	"github.com/diamondburned/aqours/internal/mpris"
	"github.com/diamondburned/aqours/internal/muse"
	"github.com/diamondburned/aqours/internal/state"
	"github.com/diamondburned/aqours/internal/ui"
	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

const (
	appFlags = gio.ApplicationFlagsNone
	appID    = "com.github.diamondburned.aqours"
)

func main() {
	log.SetFlags(log.Lmicroseconds | log.Ltime)
	glib.LogUseDefaultLogger()

	var w *ui.MainWindow

	app := gtk.NewApplication(appID, appFlags)
	app.Connect("activate", func() {
		if w == nil {
			w = activate(app)
		}
		w.Window.Present()
	})

	if exitCode := app.Run(os.Args); exitCode > 0 {
		panic(fmt.Sprintf("exit status %d", exitCode))
	}
}

func activate(app *gtk.Application) *ui.MainWindow {
	ses, err := muse.NewSession()
	if err != nil {
		log.Fatalln("Failed to create mpv session:", err)
	}

	st, err := state.ReadFromFile()
	if err != nil {
		log.Printf("failed to restore state (%v); creating a new one.\n", err)
		st = state.NewState()
	}

	w, err := ui.NewMainWindow(app, ses, st)
	if err != nil {
		log.Fatalln("Failed to create main window:", err)
	}

	// Bind MPRIS.
	m, err := mpris.New()
	if err != nil {
		log.Println("Failed to bind MPRIS:", err)
	}

	// Bind window methods.
	ses.SetHandler(m.PassthroughEvents(w))
	st.OnUpdate(m.Update)

	// Start is non-blocking, as it should be when ran inside the main
	// thread.
	ses.Start()

	// TODO: add a saving spinner circle.

	// Try to save the state and all playlists every 15 seconds.
	glib.TimeoutSecondsAdd(15, func() bool {
		st.SaveState()
		w.SaveAllPlaylists()
		return true
	})

	app.ConnectShutdown(func() {
		ses.Stop()
		m.Close()

		st.SaveAll()
		st.WaitUntilSaved()
	})

	return w
}
