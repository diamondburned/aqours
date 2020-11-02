package ffprobe

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/pkg/errors"
)

func Probe(path string) (*ProbeResult, error) {
	ctx, cancel := context.WithTimeout(context.TODO(), 15*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx,
		"ffprobe",
		"-loglevel", "fatal",
		"-print_format", "json",
		"-read_intervals", "%+1us",
		"-show_format", path,
	)
	cmd.Stderr = os.Stderr

	o, err := cmd.StdoutPipe()
	if err != nil {
		return nil, errors.Wrap(err, "failed to make stdout pipe")
	}
	defer o.Close()

	if err := cmd.Start(); err != nil {
		return nil, errors.Wrap(err, "failed to start ffprobe")
	}
	defer cmd.Wait()

	var result ProbeResult

	if err := json.NewDecoder(o).Decode(&result); err != nil {
		return nil, errors.Wrap(err, "failed to parse ffprobe JSON")
	}

	return &result, nil
}

type ProbeResult struct {
	Format Format `json:"format"`
}

type Format struct {
	Filename       string `json:"filename"`
	NbStreams      int    `json:"nb_streams"`
	NbPrograms     int    `json:"nb_programs"`
	FormatName     string `json:"format_name"`
	FormatLongName string `json:"format_long_name"`
	StartTime      string `json:"start_time"`
	Duration       string `json:"duration"`
	Size           string `json:"size"`
	BitRate        string `json:"bit_rate"`
	ProbeScore     int    `json:"probe_score"`
	Tags           Tags   `json:"tags"`
}

type Tags map[string]string

func (tags *Tags) UnmarshalJSON(v []byte) error {
	var rawTags = map[string]string{}

	if err := json.Unmarshal(v, &rawTags); err != nil {
		return err
	}

	for k, v := range rawTags {
		rawTags[strings.ToLower(k)] = v
	}

	*tags = rawTags
	return nil
}
