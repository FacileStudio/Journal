package sourcemaps

import (
	"context"
	"path"
	"regexp"
	"strconv"
	"strings"
)

// maxFrames caps how much of a stack is resolved. A runaway recursion produces
// thousands of identical frames and none of them past the first few say
// anything; resolving them all would be work nobody reads.
const maxFrames = 60

// Frame is one line of a stack, before and after resolution.
type Frame struct {
	// Raw is the line exactly as the browser wrote it. It is always present:
	// it is the evidence, and a resolution is an interpretation of it.
	Raw string `json:"raw"`
	// Function, File, Line and Column are what was parsed out of Raw. File is
	// the bundle basename.
	Function string `json:"function,omitempty"`
	File     string `json:"file,omitempty"`
	Line     int    `json:"line,omitempty"`
	Column   int    `json:"column,omitempty"`
	// Source, SourceLine, SourceColumn and Name are the resolution. Resolved
	// says whether a map was found and applied, so a reader can tell "this is
	// the original location" from "this is the bundle location".
	Resolved     bool   `json:"resolved"`
	Source       string `json:"source,omitempty"`
	SourceLine   int    `json:"source_line,omitempty"`
	SourceColumn int    `json:"source_column,omitempty"`
	Name         string `json:"name,omitempty"`
}

// Stack is a resolved trace.
type Stack struct {
	Release string  `json:"release"`
	Frames  []Frame `json:"frames"`
	// Resolved counts the frames a map explained. Zero with a non-empty
	// release means the maps for that build were never uploaded, which is a
	// different problem from having no release at all.
	Resolved int `json:"resolved"`
}

// Two shapes cover every browser in use. V8 writes
//
//	at fn (https://host/a/b.js:1:2)
//	at https://host/a/b.js:1:2
//
// and SpiderMonkey and JavaScriptCore write
//
//	fn@https://host/a/b.js:1:2
//
// Anything else is kept as a raw line rather than guessed at.
var (
	v8Frame     = regexp.MustCompile(`^\s*at\s+(?:(.+?)\s+\()?(.+?):(\d+):(\d+)\)?\s*$`)
	geckoFrame  = regexp.MustCompile(`^\s*(?:(.*?)@)?(.+?):(\d+):(\d+)\s*$`)
	frameSplits = regexp.MustCompile(`\r?\n`)
)

// Resolve turns a raw stack into frames, mapping each one it can.
//
// It never fails: a stack with no release, no uploaded map or an unparseable
// line still comes back frame by frame with Raw set. A half-resolved trace is
// useful and a refusal is not.
func (s *Service) Resolve(ctx context.Context, app, release, stack string) Stack {
	out := Stack{Release: release, Frames: []Frame{}}
	if strings.TrimSpace(stack) == "" {
		return out
	}

	for i, line := range frameSplits.Split(stack, -1) {
		if strings.TrimSpace(line) == "" {
			continue
		}
		if i >= maxFrames {
			break
		}

		frame := parseFrame(line)
		if release != "" && frame.File != "" {
			s.apply(ctx, app, release, &frame)
			if frame.Resolved {
				out.Resolved++
			}
		}
		out.Frames = append(out.Frames, frame)
	}
	return out
}

func (s *Service) apply(ctx context.Context, app, release string, frame *Frame) {
	consumer := s.consumerFor(ctx, app, release, frame.File)
	if consumer == nil {
		return
	}
	source, name, line, column, ok := consumer.Source(frame.Line, frame.Column)
	if !ok || source == "" {
		return
	}
	frame.Resolved = true
	frame.Source = tidySource(source)
	frame.SourceLine = line
	frame.SourceColumn = column
	if name != "" {
		frame.Name = name
	}
}

// parseFrame extracts what it can, and reports the line unchanged otherwise.
func parseFrame(line string) Frame {
	frame := Frame{Raw: strings.TrimRight(line, "\r")}

	match := v8Frame.FindStringSubmatch(line)
	if match == nil {
		match = geckoFrame.FindStringSubmatch(line)
	}
	if match == nil {
		return frame
	}

	lineNumber, err := strconv.Atoi(match[3])
	if err != nil {
		return frame
	}
	column, err := strconv.Atoi(match[4])
	if err != nil {
		return frame
	}

	frame.Function = strings.TrimSpace(match[1])
	frame.File = bundleName(match[2])
	frame.Line = lineNumber
	frame.Column = column
	return frame
}

// bundleName reduces a frame's location to the basename a map is keyed on.
//
// The location arrives as a full URL, and its origin and path depend on how the
// app happens to be served — which differs between a laptop, a preview and
// production, while the map is the same file. Vite hashes every emitted name,
// so the basename identifies it uniquely inside a release.
func bundleName(location string) string {
	location = strings.TrimSpace(location)
	if location == "" {
		return ""
	}
	if cut := strings.IndexAny(location, "?#"); cut >= 0 {
		location = location[:cut]
	}
	name := path.Base(location)
	if name == "." || name == "/" {
		return ""
	}
	return name
}

// tidySource strips the webpack/vite scheme prefixes a map carries in its
// sources, so a reader sees src/lib/Cart.svelte rather than a URL-ish path
// nothing can be done with.
func tidySource(source string) string {
	for _, prefix := range []string{"webpack://", "webpack-internal:///", "vite://"} {
		source = strings.TrimPrefix(source, prefix)
	}
	source = strings.TrimPrefix(source, "file://")
	if cut := strings.Index(source, "/./"); cut >= 0 {
		source = source[cut+3:]
	}
	return strings.TrimPrefix(source, "./")
}
