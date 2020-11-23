package playlist

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-test/deep"
)

type playlistTest struct {
	name   string
	apply  func(t *testing.T, pl *Playlist)
	expect []*Track
}

func TestAdd(t *testing.T) {
	testRunPlaylistTests(t, []playlistTest{
		{
			name: "empty addition",
			apply: func(t *testing.T, pl *Playlist) {
				addAndAssert(t, pl, 0, true, "0")
			},
			expect: emptyTracks("0"),
		},
		{
			name: "before",
			apply: func(t *testing.T, pl *Playlist) {
				pl.Tracks = emptyTracks("1", "2")
				addAndAssert(t, pl, 0, true, "0")
				addAndAssert(t, pl, 2, true, "1.5")
			},
			expect: emptyTracks("0", "1", "1.5", "2"),
		},
		{
			name: "before variadic",
			apply: func(t *testing.T, pl *Playlist) {
				pl.Tracks = emptyTracks("1", "2")
				addAndAssert(t, pl, 0, true, "0", "0.5")
				addAndAssert(t, pl, 3, true, "1.5", "1.75")
			},
			expect: emptyTracks("0", "0.5", "1", "1.5", "1.75", "2"),
		},
		{
			name: "after",
			apply: func(t *testing.T, pl *Playlist) {
				pl.Tracks = emptyTracks("0", "3")
				addAndAssert(t, pl, 0, false, "1")
				addAndAssert(t, pl, 1, false, "2")
				addAndAssert(t, pl, 3, false, "5")
			},
			expect: emptyTracks("0", "1", "2", "3", "5"),
		},
		{
			name: "after variadic",
			apply: func(t *testing.T, pl *Playlist) {
				pl.Tracks = emptyTracks("0", "3")
				addAndAssert(t, pl, 0, false, "1", "1.5")
				addAndAssert(t, pl, 2, false, "2", "2.5")
				addAndAssert(t, pl, 5, false, "4", "5")
			},
			expect: emptyTracks("0", "1", "1.5", "2", "2.5", "3", "4", "5"),
		},
	})
}

func addAndAssert(t *testing.T, pl *Playlist, ix int, before bool, paths ...string) {
	t.Helper()

	i, j := pl.Add(ix, before, paths...)
	assertTracks(t, pl.Tracks[i:j], emptyTracks(paths...))
}

func TestRemove(t *testing.T) {
	testRunPlaylistTests(t, []playlistTest{
		{
			name: "remove 0",
			apply: func(t *testing.T, pl *Playlist) {
				pl.Tracks = emptyTracks("0")
				pl.Remove(0)
			},
			expect: emptyTracks(),
		},
		{
			name: "between",
			apply: func(t *testing.T, pl *Playlist) {
				pl.Tracks = emptyTracks("1", "2", "3")
				pl.Remove(1)
			},
			expect: emptyTracks("1", "3"),
		},
		{
			name: "multiple between + last",
			apply: func(t *testing.T, pl *Playlist) {
				pl.Tracks = emptyTracks("1", "2", "3", "4")
				pl.Remove(2, 3)
			},
			expect: emptyTracks("1", "2"),
		},
		{
			name: "multiple first + last",
			apply: func(t *testing.T, pl *Playlist) {
				pl.Tracks = emptyTracks("1", "2", "3", "4")
				pl.Remove(0, 3)
			},
			expect: emptyTracks("2", "3"),
		},
		{
			name: "multiple first + between",
			apply: func(t *testing.T, pl *Playlist) {
				pl.Tracks = emptyTracks("1", "2", "3", "4")
				pl.Remove(0, 1, 2)
			},
			expect: emptyTracks("4"),
		},
		{
			name: "all",
			apply: func(t *testing.T, pl *Playlist) {
				pl.Tracks = emptyTracks("0", "1", "2", "3")
				pl.Remove(0, 1, 2, 3)
			},
			expect: emptyTracks(),
		},
	})
}

func testRunPlaylistTests(t *testing.T, tests []playlistTest) {
	t.Helper()

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			pl := Playlist{}
			test.apply(t, &pl)

			if !pl.IsUnsaved() {
				t.Error("playlist is unsaved after applying")
			}

			assertTracks(t, pl.Tracks, test.expect)
		})
	}
}

func assertTracks(t *testing.T, tracksGot, tracksExpected []*Track) {
	t.Helper()

	if ineqs := deep.Equal(tracksGot, tracksExpected); ineqs != nil {
		t.Errorf("got:      %s", fmtTracks(tracksGot))
		t.Errorf("expected: %s", fmtTracks(tracksExpected))
	}
}

func fmtTracks(tracks []*Track) string {
	var builder strings.Builder
	for _, track := range tracks {
		fmt.Fprintf(&builder, "%q ", track.Filepath)
	}
	return builder.String()
}

func emptyTracks(paths ...string) []*Track {
	var tracks = make([]*Track, len(paths))
	for i, path := range paths {
		tracks[i] = &Track{
			Title:    filepath.Base(path),
			Filepath: path,
		}
	}
	return tracks
}
