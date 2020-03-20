package fuzz

import (
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dapperlabs/cadence"
)

const crashersDir = "../../../crashers"

func TestCrashers(t *testing.T) {

	f, err := os.Open(crashersDir)
	if err != nil {
		t.Skip()
	}

	files, err := f.Readdir(-1)
	_ = f.Close()

	for _, file := range files {

		name := file.Name()
		if path.Ext(name) != "" {
			continue
		}

		t.Run(name, func(t *testing.T) {

			var data []byte
			data, err = ioutil.ReadFile(path.Join(crashersDir, name))
			if err != nil {
				t.Fatal(err)
			}

			assert.NotPanics(t,
				func() {
					cadence.Fuzz(data)
				},
				string(data),
			)
		})

	}
}
