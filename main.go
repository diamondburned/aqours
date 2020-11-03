package main

import (
	"log"
	"os"

	"github.com/diamondburned/aqours/internal/muse"
	"github.com/diamondburned/aqours/internal/muse/metadata/ffmpeg"
	"github.com/diamondburned/aqours/internal/ui"
	"github.com/diamondburned/handy"
	"github.com/gotk3/gotk3/gtk"
)

func main() {
	s, err := muse.NewSession()
	if err != nil {
		log.Fatalln("Failed to create mpv session:", err)
	}

	a, err := gtk.ApplicationNew("com.github.diamondburned.aqous", 0)
	if err != nil {
		log.Fatalln("Failed to create a GtkApplication:", err)
	}

	a.Connect("activate", func() {
		handy.Init()

		w, err := ui.NewMainWindow(a, s)
		if err != nil {
			log.Fatalln("Failed to create main window:", err)
		}

		// Start is non-blocking, as it should be when ran inside the main
		// thread.
		s.Start()

		w.Show()
		a.AddWindow(w)

		w.Connect("destroy", func() {
			s.Stop()
			ffmpeg.StopAll()
		})
	})

	if exitCode := a.Run(os.Args); exitCode > 0 {
		os.Exit(exitCode)
	}
}
