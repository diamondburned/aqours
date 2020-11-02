package seekbufio

// import (
// 	"bytes"
// 	"io"

// 	"github.com/pkg/errors"
// )

// // Reader buffers the first n bytes to allow quicker seeking.
// type Reader struct {
// 	prefix *bytes.Reader
// 	seeker io.ReadSeeker
// 	cursor int64
// }

// var _ io.ReadSeeker = (*Reader)(nil)

// func NewReaderSize(r io.ReadSeeker, prefixLen int64) (*Reader, error) {
// 	prefix := bytes.Buffer{}
// 	prefix.Grow(int(prefixLen))

// 	if _, err := io.CopyN(&prefix, r, prefixLen); err != nil {
// 		return nil, errors.Wrap(err, "failed to read prefix")
// 	}

// 	if _, err := r.Seek(0, io.SeekStart); err != nil {
// 		return nil, errors.Wrap(err, "failed to seek back")
// 	}

// 	return &Reader{
// 		prefix: bytes.NewReader(prefix.Bytes()),
// 		seeker: r,
// 	}, nil
// }

// func (r *Reader) Read(b []byte) (n int, err error) {
// 	n, err = r.prefix.Read(b)
// 	r.cursor += int64(n)

// 	if n == len(b) && err == nil {
// 		return n, err
// 	}

// 	r.seeker.Seek(r.cursor, io.SeekStart)

// 	return r.seeker.Read(b[n:])
// }

// func (r *Reader) Seek(offset int64, whence int) (int64, error) {
// 	n, err := r.prefix.Seek(offset, whence)
// 	if err != nil {
// 		return 0, err
// 	}

// 	r.cursor = n

// 	if int(r.cursor) > r.prefix.Len() {
// 		n, err = r.seeker.Seek(n, io.SeekStart)
// 	}

// 	return n, err
// }
