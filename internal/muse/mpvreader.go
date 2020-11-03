package muse

import (
	"bufio"
	"io"
	"log"
	"regexp"
)

// mpvReader reads the stderr log for important events.
type mpvReader struct {
	wp    *io.PipeWriter
	rp    *io.PipeReader
	log   *log.Logger
	match map[mpvLineEvent]*regexp.Regexp
}

func newMpvReader(output io.Writer, match map[mpvLineEvent]*regexp.Regexp) *mpvReader {
	rp, wp := io.Pipe()
	return &mpvReader{
		wp,
		rp,
		log.New(output, "[mpv] ", log.LstdFlags),
		match,
	}
}

func (r *mpvReader) Start(callback func(name mpvLineEvent, matches []string)) {
	go func() {
		var scanner = bufio.NewScanner(r.rp)
		for scanner.Scan() {
			// Log anyway.
			r.log.Println(scanner.Text())

			for name, regex := range r.match {
				ms := regex.FindStringSubmatch(scanner.Text())
				if ms != nil {
					callback(name, ms)
				}
			}
		}
	}()
}

func (r *mpvReader) Write(b []byte) (int, error) {
	return r.wp.Write(b)
}

func (r *mpvReader) Close() error {
	r.wp.Close()
	r.rp.Close()
	return nil
}
