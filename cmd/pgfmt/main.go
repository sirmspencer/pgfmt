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
	"path/filepath"
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
// learn: later with *fix and *check. os.Exit ends the process at once without running deferred
// learn: calls, which is why the real work lives in run() and main only passes back its exit code.
func main() {
	fix := flag.Bool("fix", false, "format each file and remove stray commas, rewriting it in place")
	check := flag.Bool("check", false, "report unformatted files, fixable problems, and syntax errors without changing anything")
	configPath := flag.String("config", "", "settings file to use instead of searching for "+configName)
	showVersion := flag.Bool("version", false, "print the version and exit")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: pgfmt [-check | -fix | -version] [-config file] [file ...]")
		fmt.Fprintln(os.Stderr, "with no files, pgfmt formats stdin to stdout")
		flag.PrintDefaults()
	}
	flag.Parse()
	if *showVersion {
		fmt.Println("pgfmt " + currentVersion())
		return
	}
	if *fix && *check {
		fmt.Fprintln(os.Stderr, "pgfmt: -check and -fix cannot be used together")
		os.Exit(2)
	}
	os.Exit(run(flag.Args(), *fix, *check, &settings{explicit: *configPath, found: map[string]format.Options{}}))
}

// exit statuses
const (
	statusClean   = 0 // nothing to change
	statusChanges = 1 // -check found formatting or fixable problems
	statusErrors  = 2 // a syntax error that cannot be fixed, or a file error
)

// learn: `fix, check bool` declares two parameters of the same type. io.ReadAll(os.Stdin) reads
// learn: until end of input. bytes.Equal compares byte slices; == does not work on slices in go.
func run(files []string, fix, check bool, config *settings) int {
	if len(files) == 0 {
		input, err := io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintln(os.Stderr, "pgfmt:", err)
			return statusErrors
		}
		cwd, _ := os.Getwd()
		opts, err := config.forDir(cwd)
		if err != nil {
			fmt.Fprintln(os.Stderr, "pgfmt:", err)
			return statusErrors
		}
		if check {
			return checkFile("<stdin>", input, opts)
		}
		// editors pipe the buffer through here: on an error nothing reaches stdout
		result, err := format.FormatResultWith(string(input), opts)
		report(os.Stderr, "<stdin>", result, err, err == nil)
		if err != nil {
			return statusErrors
		}
		os.Stdout.Write([]byte(result.Output))
		return statusClean
	}

	status := statusClean
	for _, path := range files {
		input, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintln(os.Stderr, "pgfmt:", err)
			status = statusErrors
			continue
		}
		opts, err := config.forFile(path)
		if err != nil {
			fmt.Fprintln(os.Stderr, "pgfmt:", err)
			status = statusErrors
			continue
		}
		if check {
			status = max(status, checkFile(path, input, opts))
			continue
		}
		result, err := format.FormatResultWith(string(input), opts)
		// without -fix the file is not changed, so fixes are only possible
		report(os.Stderr, path, result, err, fix && err == nil)
		if err != nil {
			status = statusErrors
			continue
		}
		output := []byte(result.Output)
		if !fix {
			os.Stdout.Write(output)
			continue
		}
		if !bytes.Equal(input, output) {
			if err := os.WriteFile(path, output, 0o644); err != nil {
				fmt.Fprintln(os.Stderr, "pgfmt:", err)
				status = statusErrors
			}
		}
	}
	return status
}

// checkFile reports everything -fix would change or could not fix, on stdout
func checkFile(name string, input []byte, opts format.Options) int {
	result, err := format.FormatResultWith(string(input), opts)
	report(os.Stdout, name, result, err, false)
	if err != nil {
		return statusErrors
	}
	if result.Output != string(input) {
		fmt.Printf("%s: not formatted\n", name)
		return statusChanges
	}
	return statusClean
}

// report prints fixes, a syntax error, and statements left unformatted, one per line
// learn: io.Writer is an interface with one method, Write. os.Stdout and os.Stderr both satisfy it,
// learn: so the caller picks where the report goes.
func report(out io.Writer, name string, result *format.Result, err error, applied bool) {
	state := "fixable"
	if applied {
		state = "fixed"
	}
	if result != nil {
		for _, fixed := range result.Fixed {
			fmt.Fprintf(out, "%s:%d:%d: %s (%s)\n", name, fixed.Line, fixed.Column, fixed.Message, state)
		}
		for _, skipped := range result.Skipped {
			fmt.Fprintf(os.Stderr, "%s:%d: left unformatted: %s\n", name, skipped.Line, skipped.Reason)
		}
	}
	if err != nil {
		var syntax *format.SyntaxError
		if errors.As(err, &syntax) {
			fmt.Fprintf(out, "%s:%d:%d: %s\n", name, syntax.Line, syntax.Column, syntax.Message)
		} else {
			fmt.Fprintf(out, "%s: %v\n", name, err)
		}
	}
}

// configName is the settings file pgfmt looks for
const configName = ".pgfmt.json"

// settings finds and caches the options for each directory
// learn: a map value is cached per directory, so a run over many files reads each config once.
type settings struct {
	explicit string
	found    map[string]format.Options
}

func (config *settings) forFile(path string) (format.Options, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return format.Options{}, err
	}
	return config.forDir(filepath.Dir(abs))
}

// forDir uses -config when given, otherwise the nearest .pgfmt.json in dir or
// a parent directory, otherwise the defaults
func (config *settings) forDir(dir string) (format.Options, error) {
	if config.explicit != "" {
		return load(config.explicit)
	}
	if opts, ok := config.found[dir]; ok {
		return opts, nil
	}
	opts := format.DefaultOptions()
	candidate := filepath.Join(dir, configName)
	switch _, err := os.Stat(candidate); {
	case err == nil:
		loaded, err := load(candidate)
		if err != nil {
			return opts, err
		}
		opts = loaded
	case filepath.Dir(dir) != dir:
		parent, err := config.forDir(filepath.Dir(dir))
		if err != nil {
			return opts, err
		}
		opts = parent
	}
	config.found[dir] = opts
	return opts, nil
}

func load(path string) (format.Options, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return format.Options{}, err
	}
	opts, err := format.ParseOptions(data)
	if err != nil {
		return opts, fmt.Errorf("%s: %w", path, err)
	}
	return opts, nil
}
