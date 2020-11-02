package audpl

import (
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/diamondburned/aqours/internal/muse/playlist"
	"github.com/diamondburned/audpl"
)

func init() {
	playlist.RegisterParser(".audpl", Parse)
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
