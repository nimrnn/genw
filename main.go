package main

import (
	"fmt"
	"bufio"
	"flag"
	"os"
	"strings"
	"net/url"
	"log/slog"
	"errors"
	"io"
)

type Options struct {
	Input		string
	Output		string
}

func NewLogger(w io.Writer) *slog.Logger {
	return slog.New(slog.NewTextHandler(w, &slog.HandlerOptions{Level: slog.LevelInfo}))
}

func ParseFlags(args []string, stderr io.Writer) (Options, error) {
	var o Options

	fs := flag.NewFlagSet("", flag.ContinueOnError)
	fs.SetOutput(stderr)

	fs.StringVar(&o.Input, "input", "", "file with one URL per line (default: read stdin")
	fs.StringVar(&o.Output, "output", "wordlist.txt", "file name to save output (default: wordlist.txt)")

	fs.Usage = func() {
		fmt.Fprintf(stderr, "usage: ./genw [flags]\n\n")
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		return Options{}, err
	}

	return o, nil
}

const maxLine = 1 << 20

func LoadSource(r io.Reader) ([]string, error) {
	var read []string
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), maxLine)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		read = append(read, line)
	}

	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("read input: %w", err)
	}

	return read, nil
}

func ExtractParams(rawUrl string) ([]string, error) {
	var params []string
	
	u, err := url.Parse(rawUrl)
	if err != nil {
		return nil, fmt.Errorf("parsing url: %w", err)
	}

	query, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		return nil, fmt.Errorf("parsing query parameters from %q: %w", rawUrl, err)
	}

	for key := range query {
		params = append(params, key)
	}

	return params, nil
}

func ExtractPaths(rawUrl string) ([]string, error) {
	u, err := url.Parse(rawUrl)
	if err != nil {
		return nil, fmt.Errorf("parsing url: %w", err)
	}

	p := strings.Trim(u.Path, "/")

	if p != "" {
		paths := strings.Split(p, "/")
		return paths, nil
	}

	return nil, nil
}

func LoadExisting(path string) (map[string]bool, error) {
	seen := make(map[string]bool)
	
	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return seen, nil
	}
	
	if err != nil {
		return nil, fmt.Errorf("opening file: %w", err)
	}

	defer file.Close()

	sc := bufio.NewScanner(file)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		seen[line] = true
	}

	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}

	return seen, nil
}

func run(args []string, stdout, stderr io.Writer) error {
	o, err := ParseFlags(args, stderr)
	if err != nil {
		return err
	}

	log := NewLogger(stderr)

	input := io.Reader(os.Stdin)
	if o.Input != "" {
		file, err := os.Open(o.Input)
		if err != nil {
			return fmt.Errorf("opening file: %w", err)
		}
		defer file.Close()
		input = file
	}

	urls, err := LoadSource(input)
	if err != nil {
		return err
	}

	if len(urls) == 0 {
		return errors.New("no urls to read")
	}

	log.Info("extracting parameters and paths", "urls", len(urls))

	seen := make(map[string]bool)
	var wordlist []string
	for _, u := range urls {
		params, err := ExtractParams(u)
		if err != nil {
			return err
		}

		for _, param := range params {
			if seen[param] {
				continue
			}
			seen[param] = true
			wordlist = append(wordlist, param)
		}

		paths, err := ExtractPaths(u)
		if err != nil {
			return err
		}

		for _, path := range paths {
			if seen[path] {
				continue
			}
			seen[path] = true
			wordlist = append(wordlist, path)
		}
	}

	for _, w := range wordlist {
		fmt.Println(w)
		
		if o.Output != "" {
			existing, err := LoadExisting(o.Output)
			if err != nil {
				return err
			}

			file, err := os.OpenFile(o.Output, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				return err
			}
			defer file.Close()
			
			if existing[w] {
				continue
			}
			existing[w] = true
			if _, err := fmt.Fprintln(file, w); err != nil {
				return fmt.Errorf("write %q: %w", w, err)
			}
		}
	}
	
	return nil
}

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
