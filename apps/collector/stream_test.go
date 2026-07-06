package main

import (
	"encoding/binary"
	"reflect"
	"testing"
)

type emitted struct {
	stream byte
	line   string
}

func frame(stream byte, payload string) []byte {
	header := make([]byte, 8)
	header[0] = stream
	binary.BigEndian.PutUint32(header[4:], uint32(len(payload)))
	return append(header, payload...)
}

func TestDemuxer(t *testing.T) {
	cases := []struct {
		name   string
		chunks [][]byte
		want   []emitted
	}{
		{
			name:   "single stdout frame",
			chunks: [][]byte{frame(1, "hello\n")},
			want:   []emitted{{streamStdout, "hello"}},
		},
		{
			name:   "stderr frame",
			chunks: [][]byte{frame(2, "boom\n")},
			want:   []emitted{{streamStderr, "boom"}},
		},
		{
			name:   "two frames in one read",
			chunks: [][]byte{append(frame(1, "a\n"), frame(2, "b\n")...)},
			want:   []emitted{{streamStdout, "a"}, {streamStderr, "b"}},
		},
		{
			name: "frame split mid header",
			chunks: [][]byte{
				frame(1, "split\n")[:3],
				frame(1, "split\n")[3:],
			},
			want: []emitted{{streamStdout, "split"}},
		},
		{
			name: "frame split mid payload",
			chunks: [][]byte{
				frame(2, "partial line\n")[:12],
				frame(2, "partial line\n")[12:],
			},
			want: []emitted{{streamStderr, "partial line"}},
		},
		{
			name: "line split across frames",
			chunks: [][]byte{
				frame(1, "first half "),
				frame(1, "second half\n"),
			},
			want: []emitted{{streamStdout, "first half second half"}},
		},
		{
			name:   "multiple lines in one frame",
			chunks: [][]byte{frame(1, "one\ntwo\n")},
			want:   []emitted{{streamStdout, "one"}, {streamStdout, "two"}},
		},
		{
			name: "interleaved streams keep separate partials",
			chunks: [][]byte{
				frame(1, "out "),
				frame(2, "err line\n"),
				frame(1, "done\n"),
			},
			want: []emitted{{streamStderr, "err line"}, {streamStdout, "out done"}},
		},
		{
			name:   "trailing partial flushed at eof",
			chunks: [][]byte{frame(1, "no newline")},
			want:   []emitted{{streamStdout, "no newline"}},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var got []emitted
			d := newDemuxer(func(stream byte, line []byte) {
				got = append(got, emitted{stream, string(line)})
			})
			for _, chunk := range tc.chunks {
				d.feed(chunk)
			}
			d.flush()
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestLineBufferRawTTY(t *testing.T) {
	cases := []struct {
		name   string
		chunks []string
		want   []string
	}{
		{
			name:   "line split across reads",
			chunks: []string{"hel", "lo\nworld\n"},
			want:   []string{"hello", "world"},
		},
		{
			name:   "partial flushed at eof",
			chunks: []string{"dangling"},
			want:   []string{"dangling"},
		},
		{
			name:   "empty line emitted",
			chunks: []string{"\nnext\n"},
			want:   []string{"", "next"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var got []string
			lb := &lineBuffer{stream: streamStdout, emit: func(stream byte, line []byte) {
				if stream != streamStdout {
					t.Fatalf("raw stream must be stdout, got %d", stream)
				}
				got = append(got, string(line))
			}}
			for _, chunk := range tc.chunks {
				lb.feed([]byte(chunk))
			}
			lb.flush()
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
	t.Run("empty line in middle of demux payload", func(t *testing.T) {
		var got []string
		lb := &lineBuffer{stream: streamStdout, emit: func(_ byte, line []byte) {
			got = append(got, string(line))
		}}
		lb.feed([]byte("a\n\nb\n"))
		lb.flush()
		want := []string{"a", "", "b"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got %v, want %v", got, want)
		}
	})
}

func TestLineBufferCapsRunawayLine(t *testing.T) {
	var got []string
	lb := &lineBuffer{stream: streamStdout, emit: func(_ byte, line []byte) {
		got = append(got, string(line))
	}}
	chunk := make([]byte, 8*1024)
	for i := range chunk {
		chunk[i] = 'x'
	}
	for i := 0; i < 10; i++ {
		lb.feed(chunk)
	}
	if len(got) == 0 {
		t.Fatal("expected capped emission for line exceeding maxLineBytes")
	}
	if len(got[0]) < maxLineBytes {
		t.Fatalf("capped line too short: %d", len(got[0]))
	}
}

func TestDemuxerZeroLengthFrame(t *testing.T) {
	var got []emitted
	d := newDemuxer(func(stream byte, line []byte) {
		got = append(got, emitted{stream, string(line)})
	})
	d.feed(frame(1, ""))
	d.feed(frame(1, "after\n"))
	want := []emitted{{streamStdout, "after"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}
