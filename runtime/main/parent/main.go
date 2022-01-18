package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"os/exec"

	"github.com/onflow/cadence/runtime"
)


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

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.Fatal(err)
	}

	if err = cmd.Start(); err != nil {
		log.Fatal(err)
	}

	// Read std-out
	stdoutString := readStdIO(stdout)

	// Read std-err
	stderrString := readStdIO(stderr)

	// 'Wait()' would close stdin, stdout and stderr pipes.
	// Therefore, no need to close them explicitly.
	err = cmd.Wait()

	// Program completed successfully.
	if err == nil {
		fmt.Print(stdoutString)
		// TODO: ideally shouldn't have anything on stderr
		return
	}

	// Program terminated with an error.
	exitError, ok := err.(*exec.ExitError)
	if ok && exitError.Exited() && exitError.ExitCode() == 1 {
		// These are the gracefully handled errors.
		// They exit with an exit-code 1 and the error content is serialized to stderr.
		// TODO: deserialize errors?
		err = runtime.Error{
			Err: errors.New(stderrString),
		}
	} else {
		// Unhandled errors.Could be
		//   - Fatal panics (e.g: stack-overflow)
		//   - Signal interrupts
		//   - etc.

		// TODO: get context?
		err = runtime.Error{
			Err: errors.New(stderrString),
		}
	}

	fmt.Print(err)
}

func readStdIO(ioReader io.ReadCloser) string {
	var buf bytes.Buffer
	buf.ReadFrom(ioReader)
	return buf.String()
}
