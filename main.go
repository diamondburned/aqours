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

		w.Show()
		a.AddWindow(w)
	})

	if err := s.Start(); err != nil {
		log.Fatalln("Failed to start mpv:", err)
	}

	defer s.Stop()
	defer ffmpeg.StopAll()

	if exitCode := a.Run(os.Args); exitCode > 0 {
		os.Exit(exitCode)
	}
}
