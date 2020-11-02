package ffmpeg

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"
)

const bufSz = 1000 * 1000 // 1MB

var globalCtx, globalStop = context.WithCancel(context.Background())

func StopAll() {
	globalStop()
}

func AlbumArt(w io.Writer, path string, size int) error {
	ctx, cancel := context.WithTimeout(globalCtx, 1*time.Minute)
	defer cancel()

	vf := fmt.Sprintf("scale=-1:'min(%d,ih)'", size)

	cmd := exec.CommandContext(ctx,
		"ffmpeg",
		"-hide_banner", "-threads", "1", "-loglevel", "error", "-y",
		"-i", path,
		"-an",
		"-c:v", "mjpeg", "-sws_flags", "lanczos", "-q:v", "5", "-vf", vf,
		"-f", "mjpeg", "-",
	)
	cmd.Stderr = os.Stderr
	cmd.Stdout = w

	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}
