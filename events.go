package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"

	"github.com/divan/gotrace/trace"
)

// EventSource defines anything that can
// return tracing events.
type EventSource interface {
	Events() ([]*trace.Event, error)
}

// TraceSource implements EventSource for
// pre-made trace file
type TraceSource struct {
	// Trace is the path to the trace file.
	Trace string
}

// NewTraceSource inits new TraceSource.
func NewTraceSource(trace string) *TraceSource {
	return &TraceSource{
		Trace: trace,
	}
}

// Events reads trace file from filesystem and symbolizes
// stacks.
func (t *TraceSource) Events() ([]*trace.Event, error) {
	f, err := os.Open(t.Trace)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	res, err := trace.Parse(f, "")
	if err != nil {
		return nil, err
	}
	return res.Events, nil

}

// NativeRun implements EventSource for running app locally,
// using native Go installation.
type NativeRun struct {
	OrigPath, Path string
}

// NewNativeRun inits new NativeRun source.
func NewNativeRun(path string) *NativeRun {
	return &NativeRun{
		OrigPath: path,
	}
}

// Events rewrites package if needed, adding Go execution tracer
// to the main function, then builds and runs package using default Go
// installation and returns parsed events.
func (r *NativeRun) Events() ([]*trace.Event, error) {
	// rewrite AST
	err := r.RewriteSource()
	if err != nil {
		return nil, fmt.Errorf("couldn't rewrite source code: %v", err)
	}
	defer func(tmpDir string) {
		if err := os.RemoveAll(tmpDir); err != nil {
			fmt.Println("Cannot remove temp dir:", err)
		}
	}(r.Path)

	tmpBinary, err := ioutil.TempFile("", "gotracer_build")
	if err != nil {
		return nil, err
	}
	defer os.Remove(tmpBinary.Name())

	// build binary
	// TODO: replace build&run part with "go run" when there is no more need
	// to keep binary
	cmd := exec.Command("go", "build", "-o", tmpBinary.Name())
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Dir = r.Path
	err = cmd.Run()
	if err != nil {
		fmt.Println("go build error", stderr.String())
		// TODO: test on most common errors, possibly add stderr to
		// error information or smth.
		return nil, err
	}

	// run
	stderr.Reset()
	cmd = exec.Command(tmpBinary.Name())
	cmd.Stderr = &stderr
	if err = cmd.Run(); err != nil {
		fmt.Println("modified program failed:", err, stderr.String())
		// TODO: test on most common errors, possibly add stderr to
		// error information or smth.
		return nil, err
	}

	if stderr.Len() == 0 {
		return nil, errors.New("empty trace")
	}

	// parse trace
	res, err := trace.Parse(&stderr, "")
	if err != nil {
		return nil, err
	}

	return res.Events, nil
}

// RewriteSource attempts to add trace-related code if needed.
// TODO: add support for multiple files and package selectors
func (r *NativeRun) RewriteSource() error {
	path, err := rewriteSource(r.OrigPath)
	if err != nil {
		return err
	}

	r.Path = path

	return nil
}

// RawSource implements EventSource for
// raw events in JSON file.
type RawSource struct {
	// Path is the path to the JSON file.
	Path string
}

// NewRawSource inits new RawSource.
func NewRawSource(path string) *RawSource {
	return &RawSource{
		Path: path,
	}
}

// Events reads JSON file from filesystem and returns events.
func (t *RawSource) Events() ([]*trace.Event, error) {
	f, err := os.Open(t.Path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var events []*trace.Event
	err = json.NewDecoder(f).Decode(&events)
	for _, ev := range events {
		fmt.Println("Event", ev)
	}
	return events, err
}
