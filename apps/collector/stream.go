package main

import (
	"bytes"
	"encoding/binary"
)

const maxLineBytes = 64 * 1024

const (
	streamStdout byte = 1
	streamStderr byte = 2
)

type lineFn func(stream byte, line []byte)

type lineBuffer struct {
	stream byte
	emit   lineFn
	buf    []byte
}

func (b *lineBuffer) feed(p []byte) {
	for len(p) > 0 {
		i := bytes.IndexByte(p, '\n')
		if i < 0 {
			b.buf = append(b.buf, p...)
			if len(b.buf) >= maxLineBytes {
				b.flush()
			}
			return
		}
		line := p[:i]
		if len(b.buf) > 0 {
			b.buf = append(b.buf, line...)
			line = b.buf
		}
		b.emit(b.stream, line)
		b.buf = b.buf[:0]
		p = p[i+1:]
	}
}

func (b *lineBuffer) flush() {
	if len(b.buf) > 0 {
		b.emit(b.stream, b.buf)
		b.buf = b.buf[:0]
	}
}

type demuxer struct {
	buf    []byte
	stdout lineBuffer
	stderr lineBuffer
}

func newDemuxer(emit lineFn) *demuxer {
	return &demuxer{
		stdout: lineBuffer{stream: streamStdout, emit: emit},
		stderr: lineBuffer{stream: streamStderr, emit: emit},
	}
}

func (d *demuxer) feed(p []byte) {
	d.buf = append(d.buf, p...)
	for {
		if len(d.buf) < 8 {
			return
		}
		size := int(binary.BigEndian.Uint32(d.buf[4:8]))
		if len(d.buf) < 8+size {
			return
		}
		payload := d.buf[8 : 8+size]
		if d.buf[0] == streamStderr {
			d.stderr.feed(payload)
		} else {
			d.stdout.feed(payload)
		}
		d.buf = d.buf[8+size:]
	}
}

func (d *demuxer) flush() {
	d.stdout.flush()
	d.stderr.flush()
}
