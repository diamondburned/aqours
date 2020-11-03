package ui

import (
	"fmt"
	"log"

	"github.com/diamondburned/aqours/internal/muse"
	"github.com/diamondburned/aqours/internal/muse/playlist"
	"github.com/diamondburned/aqours/internal/ui/content"
	"github.com/diamondburned/aqours/internal/ui/content/body/tracks"
	"github.com/diamondburned/aqours/internal/ui/css"
	"github.com/diamondburned/aqours/internal/ui/header"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
)

func init() {
	css.LoadGlobal("main", `

	`)
}

type MainWindow struct {
	gtk.ApplicationWindow

	Header  *header.Container
	Content *content.Container

	muse  *muse.Session
	state state
}

func NewMainWindow(a *gtk.Application, session *muse.Session) (*MainWindow, error) {
	w, err := gtk.ApplicationWindowNew(a)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create window")
	}
	w.SetTitle("Aqours")
	w.SetDefaultSize(800, 500)

	mw := &MainWindow{
		ApplicationWindow: *w,
		muse:              session,
		state:             newState(),
	}

	mw.Header = header.NewContainer(mw)
	mw.Header.Show()

	w.SetTitlebar(mw.Header)

	mw.Content = content.NewContainer(mw)
	mw.Content.Show()

	w.Add(mw.Content)

	session.SetHandler(mw)

	return mw, nil
}

func (w *MainWindow) OnPathUpdate(path string) {
	track := w.state.TrackList.Tracks[path]
	w.Content.Bar.NowPlaying.SetTrack(track.Track)
	w.Content.Body.Sidebar.AlbumArt.SetTrack(track.Track)
}

func (w *MainWindow) OnPauseUpdate(pause bool) {
	w.Content.Bar.Controls.Buttons.Play.SetPlaying(!pause)
}

func (w *MainWindow) OnBitrateChange(bitrate float64) {
	log.Println("Bitrate:", bitrate)
}

func (w *MainWindow) OnPositionChange(pos, total float64) {
	w.Content.Bar.Controls.Seek.UpdatePosition(pos, total)
}

func (w *MainWindow) AddPlaylist(path string) {
	w.SetSensitive(false)

	go func() {
		p, err := playlist.ParseFile(path)
		if err != nil {
			log.Println("Failed parsing playlist:", err)
		}

		glib.IdleAdd(func() {
			w.SetSensitive(true)

			if err != nil {
				return
			}

			if _, ok := w.state.Playlists[p.Name]; ok {
				log.Println("Duplicated playlist name", p.Name)
				return
			}

			uiPl := w.Content.Body.Sidebar.PlaylistList.AddPlaylist(p.Name)
			uiPl.SetTotal(len(p.Tracks))

			w.state.Playlists[p.Name] = p
		})
	}()
}

func (w *MainWindow) HasPlaylist(name string) bool {
	_, has := w.state.Playlists[name]
	return has
}

// RenamePlaylist renames a playlist. It only works if we're renaming the
// current playlist.
func (w *MainWindow) RenamePlaylist(name, newName string) bool {
	pl, ok := w.state.Playlists[name]
	if !ok {
		log.Println("Playlist not found:", name)
		return false
	}

	// Collision check.
	if _, exists := w.state.Playlists[newName]; exists {
		log.Println("Playlist's new name already exists:", newName)
		return false
	}

	pl.Name = newName
	w.state.Playlists[newName] = pl
	delete(w.state.Playlists, name)

	w.Content.Body.TracksView.DeletePlaylist(name)
	w.Content.Body.Sidebar.PlaylistList.Playlist(name).SetName(newName)
	w.SelectPlaylist(newName)

	return true
}

func (w *MainWindow) Seek(pos float64) {
	if err := w.muse.Seek(pos); err != nil {
		log.Println("Seek failed:", err)
		return
	}
}

func (w *MainWindow) Next() {
	if err := w.muse.Next(); err != nil {
		log.Println("Next failed:", err)
		return
	}
}

func (w *MainWindow) SetPlay(playing bool) {
	if err := w.muse.SetPlay(playing); err != nil {
		log.Println("SetPlay failed:", err)
		return
	}
}

func (w *MainWindow) Previous() {
	if err := w.muse.Previous(); err != nil {
		log.Println("Previous failed:", err)
		return
	}
}

func (w *MainWindow) PlayTrack(list *tracks.TrackList, n int) {
	if err := w.muse.SelectPlaylist(w.state.Playlist.Path); err != nil {
		log.Println("SelectPlaylist failed:", err)
		return
	}

	if err := w.muse.PlayTrackIndex(n); err != nil {
		log.Println("PlayTrackIndex failed:", err)
		return
	}
}

func (w *MainWindow) SelectPlaylist(name string) {
	pl, ok := w.state.Playlists[name]
	if !ok {
		log.Println("Playlist not found:", name)
		return
	}

	w.state.TrackList = w.Content.Body.TracksView.SelectPlaylist(name)
	w.state.TrackList.SetTracks(pl.Tracks)

	w.Header.SetPlaylist(name)
	w.SetTitle(fmt.Sprintf("%s - Aqours", name))
	w.state.Playlist = pl
}
