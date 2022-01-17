package main

import (
	"fmt"
	"io"
	"log"
	"os/exec"
)

const bufSize = 1024

//func main() {
//	pathToBin := "/Users/supun/work/cadence/runtime/main/main"
//	cmd := exec.Command(pathToBin, "runScript")
//
//	// cmd.Output() waits until the command finishes
//	output, err := cmd.Output()
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	fmt.Println(string(output))
//}

func main() {
	// TODO:
	//  - Handle zombie processes (parent get killed, while child still running)
	//

	pathToBin := "/Users/supun/work/cadence/runtime/main/main"
	cmd := exec.Command(pathToBin, "runScript")

	stdin, err := cmd.StdinPipe()
	if err != nil {
		log.Fatal(err)
	}
	defer closeIO(stdin)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}
	defer closeIO(stdout)

	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.Fatal(err)
	}
	defer closeIO(stderr)

	if err = cmd.Start(); err != nil {
		log.Fatal(err)
	}

	// Read std-out
	fmt.Print(readStdIO(stdout))

	// Read std-err
	fmt.Print(readStdIO(stderr))
}

func readStdIO(ioReader io.ReadCloser) string {
	var bytes = make([]byte, 0)

	for {
		buf := make([]byte, bufSize)
		n, err := ioReader.Read(buf)

		if err == io.EOF {
			break
		}

		if err != nil {
			log.Fatal(err)
		}

		bytes = append(bytes, buf...)

		if n < bufSize {
			break
		}
	}

	return string(bytes)
}

func closeIO(stdin io.Closer) {
	err := stdin.Close()
	if err != nil {
		panic(err)
	}
}
