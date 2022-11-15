package main

// Parses all programs in a CSV file with header location,code
// using am old and new runtime/cmd/parse program.
//
// It reports already broken programs, programs that are broken with the new parser,
// and when parses of the old and new parser differ

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"io"
	"log"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/go-test/deep"
)

func main() {
	if len(os.Args) < 5 {
		log.Fatal("usage: csv_path directory parse_old parse_new")
	}

	csvPath := os.Args[1]
	directory := os.Args[2]
	parseOld := os.Args[3]
	parseNew := os.Args[4]

	csvFile, err := os.Open(csvPath)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		_ = csvFile.Close()
	}()

	csvReader := csv.NewReader(csvFile)

	// Skip header
	_, _ = csvReader.Read()

	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			return
		} else if err != nil {
			log.Fatal(err)
		}

		location, code := record[0], record[1]

		compareParsing(directory, location, code, parseOld, parseNew)
	}
}

func parse(program string, path string) map[string]any {
	cmd := exec.Command(program, "-json", path)
	output, err := cmd.Output()
	if err != nil {
		if _, ok := err.(*exec.ExitError); !ok {
			log.Fatal(err)
		}
	}

	var res []any
	err = json.NewDecoder(bytes.NewReader(output)).Decode(&res)
	if err != nil {
		log.Fatal(err)
	}

	return res[0].(map[string]any)
}

func compareParsing(directory string, location string, code string, parseOld string, parseNew string) {
	log.Print(location)

	contractPath := path.Join(directory, location+".cdc")

	err := os.WriteFile(contractPath, []byte(code), 0660)
	if err != nil {
		log.Fatal(err)
	}

	res1 := parse(parseOld, contractPath)
	if parseErr, ok := res1["error"]; ok {
		log.Printf("%s is broken: %#+v", location, parseErr.(map[string]any)["Errors"])
		return
	}

	res2 := parse(parseNew, contractPath)
	if parseErr, ok := res2["error"]; ok {
		log.Printf("%s broke: %#+v", location, parseErr.(map[string]any)["Errors"])
		return
	}

	// the maximum levels of a struct to recurse into
	// this prevents infinite recursion from circular references
	deep.MaxDepth = 100

	diff := deep.Equal(res1, res2)

	if len(diff) != 0 {
		var s strings.Builder

		for _, d := range diff {
			s.WriteString(d)
			s.WriteByte('\n')
		}

		log.Printf("parses differ:\n%s", s.String())
	}
}
