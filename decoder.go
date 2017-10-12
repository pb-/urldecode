package urldecode

import (
	"encoding/hex"
	"errors"
	"io"
)

const bufferSize = 1024

// Decoder holds the decoder's internal state
type Decoder struct {
	source io.Reader
	buffer []byte
	offset int
}

// NewDecoder returns a new decoder for the given source reader
func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{
		source: r,
		buffer: make([]byte, bufferSize)[:0],
	}
}

// NextPair returns the next key/value pair in the stream or EOF if there are no more pairs
func (d *Decoder) NextPair() (string, io.Reader, error) {
	err := d.fill()
	if len(d.buffer) == 0 {
		return "", nil, err
	}

	for d.offset < len(d.buffer) {
		input := d.buffer[d.offset]
		if input == '&' {
			return string(d.buffer[:d.offset]), d.newValueReader(), nil
		} else if input == '=' {
			d.offset++
			return string(d.buffer[:d.offset-1]), d.newValueReader(), nil
		} else {
			d.offset++
		}
	}

	if d.offset < cap(d.buffer) {
		return string(d.buffer), d.newValueReader(), nil
	}

	return "", nil, errors.New("key is too long")
}

// Read decodes the value of a key/value pair
func (r *valueReader) Read(p []byte) (offset int, err error) {
	for offset < len(p) {
		// flush the output buffer if required
		if (r.flush || r.eof) && len(r.outputBuffer) > 0 {
			p[offset] = r.outputBuffer[0]
			offset++
			r.outputBuffer = r.outputBuffer[1:]
			continue
		}

		// try to read more data from the source if necessary
		if !r.eof && r.decoder.offset == len(r.decoder.buffer) {
			r.decoder.fill()
			r.eof = len(r.decoder.buffer) == 0
			continue // give us a chance to flush output buffer before returning eof
		}

		if r.eof {
			return offset, io.EOF
		}

		// we are not at EOF and ready to read the next byte
		input := r.decoder.buffer[r.decoder.offset]
		r.decoder.offset++
		if input == '&' {
			r.eof = true
		} else {
			offset = r.step(r, input, p, offset)
		}
	}

	return offset, nil
}

func (d *Decoder) newValueReader() *valueReader {
	return &valueReader{decoder: d, step: defaultStep}
}

func (d *Decoder) fill() error {
	if d.offset > 0 {
		n := copy(d.buffer, d.buffer[d.offset:])
		d.buffer = d.buffer[:n]
		d.offset = 0
	}

	// handle error later since we can have n > 0 and err != nil
	n, err := d.source.Read(d.buffer[len(d.buffer):cap(d.buffer)])
	d.buffer = d.buffer[0 : len(d.buffer)+n]

	return err
}

type valueReader struct {
	decoder      *Decoder
	eof          bool
	memory       [3]byte
	outputBuffer []byte
	step         scanStep
	flush        bool
}

type scanStep func(*valueReader, byte, []byte, int) int

// defaultStep is the normal state where we copy things verbatim (and handle +)
func defaultStep(r *valueReader, input byte, p []byte, offset int) int {
	if input == '%' {
		r.step = escape
		r.flush = false
		r.memory[0] = '%'
		r.outputBuffer = r.memory[:1]
		return offset
	}

	if input == '+' {
		p[offset] = ' '
	} else {
		p[offset] = input
	}

	return offset + 1
}

// escape is the state after reading %
func escape(r *valueReader, input byte, p []byte, offset int) int {
	r.memory[1] = input
	r.outputBuffer = r.memory[:2]
	if input >= '0' && input <= '9' || input >= 'a' && input <= 'f' || input >= 'A' && input <= 'F' {
		r.step = escape1
	} else {
		r.step = defaultStep
		r.flush = true
	}

	return offset
}

// escape is the state after reading %x where x is a hex digit
func escape1(r *valueReader, input byte, p []byte, offset int) int {
	r.memory[2] = input
	r.outputBuffer = r.memory[:3]
	r.step = defaultStep
	r.flush = true
	if input >= '0' && input <= '9' || input >= 'a' && input <= 'f' || input >= 'A' && input <= 'F' {
		hex.Decode(p[offset:offset+1], r.outputBuffer[1:])
		r.outputBuffer = nil
		return offset + 1
	}

	return offset
}
