package ffprobe

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"strconv"
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
		"-show_format",
		"-show_streams", "-select_streams", "a:0",
		path,
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
	Format  Format   `json:"format"`
	Streams []Stream `json:"streams"`
}

var escaper = strings.NewReplacer("\n", `↵`)

// TagValue searches the given key name in all possible tags in the format and
// all streams.
func (res ProbeResult) TagValue(name string) string {
	if v, ok := res.Format.Tags[name]; ok {
		return escaper.Replace(v)
	}

	for _, stream := range res.Streams {
		if v, ok := stream.Tags[name]; ok {
			return escaper.Replace(v)
		}
	}

	return ""
}

func (res ProbeResult) TagValueInt(name string, orInt int) int {
	v := res.TagValue(name)
	// Split the slash for certain values like 0/3 (track number), etc.
	i, err := strconv.Atoi(strings.SplitN(v, "/", 2)[0])
	if err != nil {
		return orInt
	}
	return i
}

type Format struct {
	Duration float64 `json:"duration,string"`
	BitRate  int     `json:"bit_rate,string"`
	Tags     Tags    `json:"tags"`
}

type Stream struct {
	CodecName     string `json:"codec_name"`
	SampleRate    int    `json:"sample_rate,string"`
	Channels      int    `json:"channels"`
	ChannelLayout string `json:"channel_layout"`
	Tags          Tags   `json:"tags"`
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
