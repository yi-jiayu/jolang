package jo

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

type parserBenchmark struct {
	name   string
	source string
}

func BenchmarkParse(b *testing.B) {
	examples, err := filepath.Glob("examples/*.jo")
	if err != nil {
		b.Fatal(err)
	}
	var bms []parserBenchmark
	for _, p := range examples {
		f, err := os.Open(p)
		if err != nil {
			b.Fatal(err)
		}
		source, err := ioutil.ReadAll(f)
		bms = append(bms, parserBenchmark{
			name:   p,
			source: string(source),
		})
	}
	sort.Slice(bms, func(i, j int) bool {
		return bms[i].name < bms[j].name
	})
	for _, bm := range bms {
		b.Run(bm.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_, err := Parse(bm.source)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
