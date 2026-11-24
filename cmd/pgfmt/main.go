// Command pgfmt formats postgres sql files.
//
// With no files it reads stdin and writes the result to stdout, which is
// the mode editors use. On a syntax error nothing is written to stdout and
// the exit status is 2, so editors keep the buffer unchanged.
// learn: package main with a func main() builds an executable rather than a library. `go build
// learn: ./cmd/pgfmt` names the binary after the directory.
package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime/debug"

	"github.com/sirmspencer/pgfmt/format"
)

// version is stamped at build time, for example by the homebrew formula
// learn: -ldflags "-X main.version=v0.1.0" tells the go linker to overwrite this string variable
// learn: when building the binary. it only works on package-level string variables.
var version = ""

// currentVersion reports the stamped version, the module version for go install builds, or dev
// learn: debug.ReadBuildInfo returns metadata embedded in every go binary. after
// learn: `go install github.com/sirmspencer/pgfmt/cmd/pgfmt@v0.1.0`, Main.Version is "v0.1.0";
// learn: a local `go build` reports "(devel)" instead.
func currentVersion() string {
	if version != "" {
		return version
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return "dev"
}

// learn: flag.Bool returns a *bool that is filled in when flag.Parse() runs, so the values are read
// learn: later with *write and *check. os.Exit ends the process at once without running deferred
// learn: calls, which is why the real work lives in run() and main only passes back its exit code.
func main() {
	write := flag.Bool("w", false, "write the result back to each file instead of stdout")
	check := flag.Bool("check", false, "list files that are not formatted and exit 1 if any")
	showVersion := flag.Bool("version", false, "print the version and exit")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: pgfmt [-w | -check | -version] [file ...]")
		flag.PrintDefaults()
	}
	flag.Parse()
	if *showVersion {
		fmt.Println("pgfmt " + currentVersion())
		return
	}
	os.Exit(run(flag.Args(), *write, *check))
}

// learn: `write, check bool` declares two parameters of the same type. io.ReadAll(os.Stdin) reads
// learn: until end of input. bytes.Equal compares byte slices; == does not work on slices in go.
func run(files []string, write, check bool) int {
	if len(files) == 0 {
		input, err := io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintln(os.Stderr, "pgfmt:", err)
			return 2
		}
		output, err := formatText("<stdin>", input)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 2
		}
		if check {
			if !bytes.Equal(input, output) {
				fmt.Println("<stdin>")
				return 1
			}
			return 0
		}
		os.Stdout.Write(output)
		return 0
	}

	status := 0
	for _, path := range files {
		input, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintln(os.Stderr, "pgfmt:", err)
			status = 2
			continue
		}
		output, err := formatText(path, input)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			status = 2
			continue
		}
		switch {
		case check:
			if !bytes.Equal(input, output) {
				fmt.Println(path)
				status = max(status, 1)
			}
		case write:
			if !bytes.Equal(input, output) {
				if err := os.WriteFile(path, output, 0o644); err != nil {
					fmt.Fprintln(os.Stderr, "pgfmt:", err)
					status = 2
				}
			}
		default:
			os.Stdout.Write(output)
		}
	}
	return status
}

// formatText formats one input, reporting skipped statements on stderr
// learn: string(input) and []byte(text) convert between strings and byte slices by copying.
// learn: fmt.Errorf with %w wraps an error so errors.As and errors.Is can still find the original.
func formatText(name string, input []byte) ([]byte, error) {
	result, err := format.FormatResult(string(input))
	if err != nil {
		var syntax *format.SyntaxError
		if errors.As(err, &syntax) {
			return nil, fmt.Errorf("%s:%d:%d: %s", name, syntax.Line, syntax.Column, syntax.Message)
		}
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	for _, skipped := range result.Skipped {
		fmt.Fprintf(os.Stderr, "%s:%d: left unformatted: %s\n", name, skipped.Line, skipped.Reason)
	}
	return []byte(result.Output), nil
}
