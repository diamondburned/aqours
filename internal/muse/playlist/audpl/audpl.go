package audpl

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/diamondburned/aqours/internal/muse/playlist"
	"github.com/diamondburned/audpl"
	"github.com/pkg/errors"
)

func init() {
	playlist.Register(".audpl", Parse, Write)
}

func Parse(path string) (*playlist.Playlist, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	f.SetDeadline(time.Now().Add(15 * time.Second))

	p, err := audpl.Parse(f)
	if err != nil {
		return nil, err
	}

	var playlistCopy = playlist.Playlist{
		Name:   p.Name,
		Path:   path,
		Tracks: make([]*playlist.Track, 0, len(p.Tracks)),
	}

	for _, track := range p.Tracks {
		if !strings.HasPrefix(track.URI, "file://") {
			log.Println("[audpl]: rogue path not in local fs:", track.URI)
			continue
		}

		path := strings.TrimPrefix(track.URI, "file://")

		trackNum, _ := strconv.Atoi(track.TrackNumber)
		lengthMs, _ := strconv.Atoi(track.Length)
		bitrateKbit, _ := strconv.Atoi(track.Bitrate)

		playlistCopy.Tracks = append(playlistCopy.Tracks, &playlist.Track{
			Title:    track.Title,
			Artist:   track.Artist,
			Album:    track.Album,
			Number:   trackNum,
			Length:   time.Duration(lengthMs) * time.Millisecond,
			Bitrate:  bitrateKbit * 1000,
			Filepath: path,
		})
	}

	return &playlistCopy, nil
}

func Write(p *playlist.Playlist, done func(error)) error {
	plist := audpl.Playlist{
		Name:   p.Name,
		Tracks: make([]audpl.Track, len(p.Tracks)),
	}

	for i, track := range p.Tracks {
		plist.Tracks[i] = audpl.Track{
			Title:       track.Title,
			Artist:      track.Artist,
			Album:       track.Album,
			TrackNumber: strconv.Itoa(track.Number),
			Length:      strconv.Itoa(int(track.Length / time.Millisecond)),
			Bitrate:     strconv.Itoa(int(track.Bitrate / 1000)),
			URI:         fmt.Sprintf("file://%s", track.Filepath),
		}
	}

	go func() {
		f, err := os.Create(p.Path)
		if err != nil {
			done(errors.Wrap(err, "failed to create playlist file"))
			return
		}
		defer f.Close()

		buf := bufio.NewWriter(f)

		if err := plist.SaveTo(f); err != nil {
			done(errors.Wrap(err, "failed to write playlist"))
			return
		}

		if err := buf.Flush(); err != nil {
			done(errors.Wrap(err, "failed to flush"))
			return
		}

		done(nil)
	}()

	return nil
}
