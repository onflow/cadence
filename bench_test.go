package format_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/janezpodhostnik/cadencefmt/internal/format"
	"github.com/janezpodhostnik/cadencefmt/internal/format/render"
	"github.com/janezpodhostnik/cadencefmt/internal/format/rewrite"
	"github.com/janezpodhostnik/cadencefmt/internal/format/trivia"
	"github.com/janezpodhostnik/cadencefmt/internal/format/verify"
	"github.com/onflow/cadence/parser"
	"github.com/turbolent/prettier"
)

type corpusFile struct {
	name string
	data []byte
}

func loadSnapshotInputs(b *testing.B) map[string][]byte {
	b.Helper()
	root := findRepoRoot(b)
	dir := filepath.Join(root, "testdata", "format")
	entries, err := os.ReadDir(dir)
	if err != nil {
		b.Fatalf("reading testdata dir: %v", err)
	}
	inputs := make(map[string][]byte, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name(), "input.cdc"))
		if err != nil {
			b.Fatalf("reading input %s: %v", e.Name(), err)
		}
		inputs[e.Name()] = data
	}
	return inputs
}

func loadCorpusFiles(b *testing.B) []corpusFile {
	b.Helper()
	root := findRepoRoot(b)
	corpusDir := filepath.Join(root, "testdata", "corpus")
	if _, err := os.Stat(corpusDir); os.IsNotExist(err) {
		return nil
	}
	var files []corpusFile
	err := filepath.WalkDir(corpusDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || filepath.Ext(path) != ".cdc" {
			return err
		}
		rel, _ := filepath.Rel(corpusDir, path)
		if corpusSkip[rel] {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		files = append(files, corpusFile{rel, data})
		return nil
	})
	if err != nil {
		b.Fatalf("walking corpus dir: %v", err)
	}
	return files
}

func largestCorpusFile(b *testing.B) []byte {
	b.Helper()
	files := loadCorpusFiles(b)
	if files == nil {
		b.Skip("corpus not checked out; run: git submodule update --init")
	}
	var largest []byte
	for _, f := range files {
		if len(f.data) > len(largest) {
			largest = f.data
		}
	}
	return largest
}

// --- End-to-end benchmarks ---

func BenchmarkFormat_Snapshot(b *testing.B) {
	inputs := loadSnapshotInputs(b)
	opts := format.Default()

	var totalBytes int64
	for _, data := range inputs {
		totalBytes += int64(len(data))
	}

	b.ResetTimer()
	b.SetBytes(totalBytes)
	for b.Loop() {
		for name, data := range inputs {
			if _, err := format.Format(data, name+".cdc", opts); err != nil {
				b.Fatalf("format %s: %v", name, err)
			}
		}
	}
}

func BenchmarkFormat_PerCase(b *testing.B) {
	inputs := loadSnapshotInputs(b)
	opts := format.Default()

	for name, data := range inputs {
		b.Run(name, func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			for b.Loop() {
				if _, err := format.Format(data, name+".cdc", opts); err != nil {
					b.Fatalf("format: %v", err)
				}
			}
		})
	}
}

func BenchmarkFormat_Corpus_Small(b *testing.B)  { benchCorpusBucket(b, 0, 1024) }
func BenchmarkFormat_Corpus_Medium(b *testing.B) { benchCorpusBucket(b, 1024, 10*1024) }
func BenchmarkFormat_Corpus_Large(b *testing.B)  { benchCorpusBucket(b, 10*1024, 200*1024) }

func benchCorpusBucket(b *testing.B, minSize, maxSize int) {
	b.Helper()
	files := loadCorpusFiles(b)
	if files == nil {
		b.Skip("corpus not checked out; run: git submodule update --init")
	}

	var bucket []corpusFile
	for _, f := range files {
		if len(f.data) >= minSize && len(f.data) < maxSize {
			bucket = append(bucket, f)
		}
	}
	if len(bucket) == 0 {
		b.Skipf("no corpus files in range [%d, %d)", minSize, maxSize)
	}

	opts := format.Default()
	var totalBytes int64
	for _, f := range bucket {
		totalBytes += int64(len(f.data))
	}

	b.ResetTimer()
	b.SetBytes(totalBytes)
	for b.Loop() {
		for _, f := range bucket {
			if _, err := format.Format(f.data, f.name, opts); err != nil {
				b.Fatalf("format %s: %v", f.name, err)
			}
		}
	}
}

func BenchmarkFormat_LargestFile(b *testing.B) {
	src := largestCorpusFile(b)
	opts := format.Default()

	b.ResetTimer()
	b.SetBytes(int64(len(src)))
	for b.Loop() {
		if _, err := format.Format(src, "bench.cdc", opts); err != nil {
			b.Fatalf("format: %v", err)
		}
	}
}

// --- Per-stage benchmarks (on the largest corpus file) ---

func BenchmarkStage_Parse(b *testing.B) {
	src := largestCorpusFile(b)
	b.ResetTimer()
	b.SetBytes(int64(len(src)))
	for b.Loop() {
		if _, err := parser.ParseProgram(nil, src, parser.Config{}); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkStage_TriviaScan(b *testing.B) {
	src := largestCorpusFile(b)
	b.ResetTimer()
	b.SetBytes(int64(len(src)))
	for b.Loop() {
		trivia.Scan(src)
	}
}

func BenchmarkStage_TriviaAttach(b *testing.B) {
	src := largestCorpusFile(b)
	b.ResetTimer()
	b.SetBytes(int64(len(src)))
	for b.Loop() {
		// Re-parse each iteration: Attach builds a CommentMap tied to AST node pointers,
		// and Group/Attach don't mutate the program, but we need a fresh CommentMap.
		program, _ := parser.ParseProgram(nil, src, parser.Config{})
		comments := trivia.Scan(src)
		groups := trivia.Group(comments, src)
		trivia.Attach(program, groups, src)
	}
}

func BenchmarkStage_Rewrite(b *testing.B) {
	src := largestCorpusFile(b)
	b.ResetTimer()
	for b.Loop() {
		// rewrite.Apply mutates the AST, so re-parse each iteration.
		// Setup cost (parse + trivia) is included; rewrite itself is very fast.
		program, _ := parser.ParseProgram(nil, src, parser.Config{})
		comments := trivia.Scan(src)
		groups := trivia.Group(comments, src)
		cm := trivia.Attach(program, groups, src)
		if err := rewrite.Apply(program, cm); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkStage_Render(b *testing.B) {
	src := largestCorpusFile(b)
	b.ResetTimer()
	b.SetBytes(int64(len(src)))
	for b.Loop() {
		// render.Program consumes the CommentMap via Take(), so we need a fresh
		// CommentMap each iteration. This means re-running the trivia pipeline.
		program, _ := parser.ParseProgram(nil, src, parser.Config{})
		comments := trivia.Scan(src)
		groups := trivia.Group(comments, src)
		cm := trivia.Attach(program, groups, src)
		if err := rewrite.Apply(program, cm); err != nil {
			b.Fatal(err)
		}
		ctx := &render.Context{}
		render.Program(program, cm, ctx)
	}
}

func BenchmarkStage_PrettyPrint(b *testing.B) {
	src := largestCorpusFile(b)
	program, _ := parser.ParseProgram(nil, src, parser.Config{})
	comments := trivia.Scan(src)
	groups := trivia.Group(comments, src)
	cm := trivia.Attach(program, groups, src)
	if err := rewrite.Apply(program, cm); err != nil {
		b.Fatal(err)
	}
	ctx := &render.Context{}
	doc := render.Program(program, cm, ctx)

	var buf bytes.Buffer
	b.ResetTimer()
	for b.Loop() {
		buf.Reset()
		prettier.Prettier(&buf, doc, 100, "    ")
	}
}

func BenchmarkStage_Verify(b *testing.B) {
	src := largestCorpusFile(b)
	formatted, err := format.Format(src, "bench.cdc", format.Default())
	if err != nil {
		b.Fatalf("format: %v", err)
	}
	b.ResetTimer()
	b.SetBytes(int64(len(src)))
	for b.Loop() {
		if err := verify.RoundTrip(src, formatted); err != nil {
			b.Fatal(err)
		}
	}
}
