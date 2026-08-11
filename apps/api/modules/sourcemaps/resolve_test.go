package sourcemaps

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/FacileStudio/Journal/apps/api/internal/testdb"
)

func TestParseFrameAcrossBrowsers(t *testing.T) {
	cases := []struct {
		name string
		line string
		fn   string
		file string
		row  int
		col  int
	}{
		{
			"v8 named",
			"    at handleClick (https://sablier.facile.studio/_app/immutable/chunks/BxYz1234.js:1:48291)",
			"handleClick", "BxYz1234.js", 1, 48291,
		},
		{
			"v8 anonymous",
			"    at https://sablier.facile.studio/_app/immutable/chunks/BxYz1234.js:12:7",
			"", "BxYz1234.js", 12, 7,
		},
		{
			"gecko",
			"handleClick@https://sablier.facile.studio/_app/immutable/chunks/BxYz1234.js:1:48291",
			"handleClick", "BxYz1234.js", 1, 48291,
		},
		{
			"gecko anonymous",
			"@https://sablier.facile.studio/_app/immutable/chunks/BxYz1234.js:3:9",
			"", "BxYz1234.js", 3, 9,
		},
		{
			"query string dropped",
			"    at fn (https://host/_app/chunks/a.js?v=2:4:5)",
			"fn", "a.js", 4, 5,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			frame := parseFrame(tc.line)
			if frame.Function != tc.fn || frame.File != tc.file || frame.Line != tc.row || frame.Column != tc.col {
				t.Fatalf("parsed %+v, want fn=%q file=%q %d:%d", frame, tc.fn, tc.file, tc.row, tc.col)
			}
			if frame.Raw == "" {
				t.Fatal("Raw was dropped; it is the evidence and must always survive")
			}
		})
	}
}

// A line nothing recognises has to come back intact rather than be guessed at
// or dropped: half a stack is still worth reading.
func TestUnparseableFrameKeepsItsLine(t *testing.T) {
	frame := parseFrame("Error: something went wrong")
	if frame.File != "" || frame.Line != 0 {
		t.Fatalf("guessed at an unparseable line: %+v", frame)
	}
	if frame.Raw != "Error: something went wrong" {
		t.Fatalf("Raw = %q, want the line unchanged", frame.Raw)
	}
}

func TestTidySource(t *testing.T) {
	cases := map[string]string{
		"webpack://app/./src/lib/Cart.svelte": "src/lib/Cart.svelte",
		"./src/routes/+page.svelte":           "src/routes/+page.svelte",
		"src/lib/Cart.svelte":                 "src/lib/Cart.svelte",
		"file:///app/src/main.ts":             "/app/src/main.ts",
	}
	for input, want := range cases {
		if got := tidySource(input); got != want {
			t.Errorf("tidySource(%q) = %q, want %q", input, got, want)
		}
	}
}

// Resolving must never refuse. An entry with no release, or from an app that
// uploaded nothing, still yields its frames — otherwise adopting source maps
// would make unmapped errors *less* readable than before.
func TestResolveWithoutMapsStillReturnsFrames(t *testing.T) {
	service := NewService(nil)
	stack := "    at fn (https://host/a/b.js:1:2)\n    at other (https://host/a/c.js:3:4)"

	out := service.Resolve(context.Background(), "sablier", "", stack)

	if len(out.Frames) != 2 {
		t.Fatalf("got %d frames, want 2", len(out.Frames))
	}
	if out.Resolved != 0 {
		t.Fatalf("Resolved = %d with no release, want 0", out.Resolved)
	}
	if out.Frames[0].File != "b.js" || out.Frames[0].Line != 1 {
		t.Fatalf("frame not parsed: %+v", out.Frames[0])
	}
}

func TestResolveIgnoresBlankStack(t *testing.T) {
	service := NewService(nil)
	if out := service.Resolve(context.Background(), "app", "v1", "   \n  "); len(out.Frames) != 0 {
		t.Fatalf("got %d frames for a blank stack, want 0", len(out.Frames))
	}
}

// A render loop produces thousands of identical frames; resolving them all is
// work nobody reads.
func TestResolveCapsFrameCount(t *testing.T) {
	service := NewService(nil)
	line := "    at fn (https://host/a/b.js:1:2)\n"
	stack := ""
	for i := 0; i < maxFrames*3; i++ {
		stack += line
	}

	if out := service.Resolve(context.Background(), "app", "", stack); len(out.Frames) != maxFrames {
		t.Fatalf("got %d frames, want the cap of %d", len(out.Frames), maxFrames)
	}
}

// The end-to-end case, against a map esbuild actually produced: a minified
// position becomes the original file, line and column. Everything above this
// tests the plumbing; this tests the feature.
func TestResolveAgainstARealMap(t *testing.T) {
	db := testdb.Migrated(t)
	service := NewService(db)
	content, err := os.ReadFile(filepath.Join("testdata", "cart.js.map"))
	if err != nil {
		t.Fatalf("read the fixture: %v", err)
	}

	stored, err := service.Store(context.Background(), "sablier", "v1", "cart.js", string(content))
	if err != nil {
		t.Fatalf("Store: %v", err)
	}
	if !stored {
		t.Fatal("the first upload reported nothing stored")
	}

	// The throw sits at column 20 of the single minified line. A browser
	// reports columns 1-based, which is what a stack carries.
	stack := "    at n (https://sablier.facile.studio/_app/immutable/chunks/cart.js:1:21)"
	out := service.Resolve(context.Background(), "sablier", "v1", stack)

	if out.Resolved != 1 {
		t.Fatalf("Resolved = %d, want 1: %+v", out.Resolved, out.Frames)
	}
	frame := out.Frames[0]
	if frame.Source != "cart.js" {
		t.Errorf("Source = %q, want cart.js", frame.Source)
	}
	if frame.SourceLine != 3 {
		t.Errorf("SourceLine = %d, want 3 (the throw)", frame.SourceLine)
	}
	if !frame.Resolved {
		t.Error("the frame is not marked resolved")
	}
	if frame.Raw == "" {
		t.Error("the raw line was dropped")
	}
}

// A second upload of the same file is what every restart does. It must be
// quiet, and it must not duplicate the row.
func TestStoreIsIdempotent(t *testing.T) {
	db := testdb.Migrated(t)
	service := NewService(db)
	content, err := os.ReadFile(filepath.Join("testdata", "cart.js.map"))
	if err != nil {
		t.Fatalf("read the fixture: %v", err)
	}

	if _, err := service.Store(context.Background(), "sablier", "v1", "cart.js", string(content)); err != nil {
		t.Fatalf("first Store: %v", err)
	}
	stored, err := service.Store(context.Background(), "sablier", "v1", "cart.js", string(content))
	if err != nil {
		t.Fatalf("second Store: %v", err)
	}
	if stored {
		t.Error("a re-upload reported a new row")
	}

	files, err := service.Files(context.Background(), "sablier", "v1")
	if err != nil {
		t.Fatalf("Files: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("files = %v, want exactly one", files)
	}
}

// A map that does not parse must be refused at upload. Stored, it would look
// present and silently resolve nothing — the worst of both.
func TestStoreRejectsGarbage(t *testing.T) {
	service := NewService(testdb.Migrated(t))
	if _, err := service.Store(context.Background(), "app", "v1", "a.js", "not a source map"); err == nil {
		t.Fatal("garbage was accepted as a source map")
	}
}

// Deleting a release must also drop what the cache holds, or a rollback keeps
// resolving against the build that was rolled back.
func TestDeleteEvictsTheCache(t *testing.T) {
	db := testdb.Migrated(t)
	service := NewService(db)
	content, _ := os.ReadFile(filepath.Join("testdata", "cart.js.map"))
	ctx := context.Background()
	if _, err := service.Store(ctx, "sablier", "v1", "cart.js", string(content)); err != nil {
		t.Fatalf("Store: %v", err)
	}

	stack := "    at n (https://host/cart.js:1:21)"
	if out := service.Resolve(ctx, "sablier", "v1", stack); out.Resolved != 1 {
		t.Fatalf("the fixture did not resolve before deletion: %+v", out)
	}

	if _, err := service.Delete(ctx, "sablier", "v1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if out := service.Resolve(ctx, "sablier", "v1", stack); out.Resolved != 0 {
		t.Fatal("a deleted release still resolves from cache")
	}
}
