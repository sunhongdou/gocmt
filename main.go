package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/format"
	"go/token"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	// ensure that the comment starts on a newline (without the \n, sometimes it starts on the previous }
	commentBase = "\n// %s "
	// if it's in an indented block, this makes sure that the indentation is correct
	commentIndentedBase = "// %s "
	fset                = token.NewFileSet()
	defaultMode         = os.FileMode(0644)
	tralingTabRegex     = regexp.MustCompile(`(?m)\t+$`)
	newlinesRegex       = regexp.MustCompile(`(?m)\n{3,}`)
)

var (
	inPlace  = flag.Bool("i", false, "Make in-place editing")
	template = flag.String("t", "...", "Comment template")
	dir      = flag.String("d", "", "Directory to process")
)

func main() {
	os.Exit(gocmtRun())
}

func usage() {
	fmt.Fprintf(os.Stderr, "usage: gocmt [flags] [file ...]\n")
	flag.PrintDefaults()
}

func gocmtRun() int {
	flag.Parse()

	if *dir != "" {
		if err := filepath.Walk(*dir, walkFunc); err != nil {
			printError(err)
			return 1
		}
		return 0
	}

	if flag.NArg() == 0 {
		usage()
	}

	for i := 0; i < flag.NArg(); i++ {
		path := flag.Arg(i)
		switch fi, err := os.Stat(path); {
		case err != nil:
			printError(err)
		case fi.IsDir():
			printError(fmt.Errorf("%s is a directory", path))
		default:
			if err := processFile(path, *template, *inPlace); err != nil {
				printError(err)
				return 1
			}
		}
	}

	return 0
}

func processFile(filename, template string, inPlace bool) error {
	// skip test files
	if strings.HasSuffix(filename, "_test.go") {
		return nil
	}

	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer func() {
		closeErr := f.Close()
		if closeErr != nil {
			panic(closeErr)
		}
	}()

	orig, err := ioutil.ReadAll(f)
	if err != nil {
		return err
	}

	af, err := parseFile(fset, filename, template)
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	if err := format.Node(&buf, fset, af); err != nil {
		panic(err)
	}

	newBuf := buf.Bytes()
	if !bytes.Equal(orig, newBuf) {
		newBuf = tralingTabRegex.ReplaceAll(newBuf, []byte(""))
		newBuf = newlinesRegex.ReplaceAll(newBuf, []byte("\n\n"))

		if inPlace {
			return ioutil.WriteFile(filename, newBuf, defaultMode)
		}

		fmt.Fprintf(os.Stdout, "%s", newBuf)
		return nil
	}

	fmt.Fprintf(os.Stderr, "%s no changes\n", filename)

	return nil
}
