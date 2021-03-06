// Copyright 2018 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestStatus(t *testing.T) {
	ctx := context.Background()
	env, err := newTestEnv(ctx, t)
	if err != nil {
		t.Fatal(err)
	}
	defer env.cleanup()
	if err := stageCommitTest(ctx, env, true); err != nil {
		t.Fatal(err)
	}

	out, err := env.gg(ctx, env.root, "status")
	if err != nil {
		t.Fatal(err)
	}
	got := parseGGStatus(out, t)
	want := []ggStatusLine{
		{letter: 'A', name: "added.txt"},
		{letter: 'M', name: "modified.txt"},
		{letter: 'R', name: "deleted.txt"},
	}
	diff := cmp.Diff(want, got,
		cmp.AllowUnexported(ggStatusLine{}),
		cmp.Transformer("Map", ggStatusMap),
		cmpopts.EquateEmpty())
	if diff != "" {
		t.Errorf("Output differs (-want +got):\n%s", diff)
	}
}

// TestStatus_RenamedLocally is a regression test for
// https://github.com/zombiezen/gg/issues/44.
func TestStatus_RenamedLocally(t *testing.T) {
	ctx := context.Background()
	env, err := newTestEnv(ctx, t)
	if err != nil {
		t.Fatal(err)
	}
	defer env.cleanup()
	if err := env.git.Run(ctx, "init"); err != nil {
		t.Fatal(err)
	}
	err = ioutil.WriteFile(filepath.Join(env.root, "foo.txt"), []byte("Hello, World!\n"), 0666)
	if err != nil {
		t.Fatal(err)
	}
	if err := env.git.Run(ctx, "add", "foo.txt"); err != nil {
		t.Fatal(err)
	}
	if err := env.git.Run(ctx, "commit", "-m", "initial commit"); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(filepath.Join(env.root, "foo.txt"), filepath.Join(env.root, "bar.txt")); err != nil {
		t.Fatal(err)
	}
	if err := env.git.Run(ctx, "add", "-N", "bar.txt"); err != nil {
		t.Fatal(err)
	}

	out, err := env.gg(ctx, env.root, "status")
	if err != nil {
		t.Fatal(err)
	}
	got := parseGGStatus(out, t)
	want := []ggStatusLine{
		{letter: '!', name: "foo.txt"},
		{letter: 'A', name: "bar.txt"},
	}
	diff := cmp.Diff(want, got,
		cmp.AllowUnexported(ggStatusLine{}),
		cmp.Transformer("Map", ggStatusMap),
		cmpopts.EquateEmpty())
	if diff != "" {
		t.Errorf("Output differs (-want +got):\n%s", diff)
	}
}

type ggStatusLine struct {
	letter byte
	name   string
	from   string
}

// parseGGStatus parses the lines emitted by `gg status`, reporting any parse errors to e.
func parseGGStatus(out []byte, e errorer) []ggStatusLine {
	var lines []ggStatusLine
	for lineno, canAddFrom := 1, false; len(out) > 0; lineno++ {
		// Find end of current line.
		var line []byte
		if i := bytes.IndexByte(out, '\n'); i != -1 {
			line, out = out[:i], out[i+1:]
		} else {
			line, out = out, nil
		}

		// Validate format of line.
		if len(line) < 3 {
			e.Errorf("Line %d: got %q; want >=3 characters for status, then space, then name", lineno, line)
			canAddFrom = false
			continue
		}
		if line[1] != ' ' {
			e.Errorf("Line %d: got %q; want second character to be a space", lineno, line)
			canAddFrom = false
			continue
		}
		name := string(line[2:])

		if line[0] == ' ' {
			// Copy/rename source.
			if !canAddFrom {
				e.Errorf("Line %d: got %q (a \"from\" line); not valid with previous line", lineno, name)
				continue
			}
			lines[len(lines)-1].from = name
			canAddFrom = false
			continue
		}

		if hasGGStatusLine(lines, name) {
			e.Errorf("Line %d: duplicate for %s", lineno, name)
			canAddFrom = false
			continue
		}
		lines = append(lines, ggStatusLine{
			letter: line[0],
			name:   name,
		})
		canAddFrom = true
	}
	return lines
}

func TestParseGGStatus(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []ggStatusLine
		err  bool
	}{
		{name: "Empty", in: "", want: nil},
		{name: "SingleFile", in: "A foo.txt\n", want: []ggStatusLine{
			{letter: 'A', name: "foo.txt"},
		}},
		{name: "Copied", in: "A foo.txt\n  bar.txt\n", want: []ggStatusLine{
			{letter: 'A', name: "foo.txt", from: "bar.txt"},
		}},
		{name: "ThreeFiles", in: "A added.txt\nM modified.txt\nR deleted.txt\n", want: []ggStatusLine{
			{letter: 'A', name: "added.txt"},
			{letter: 'M', name: "modified.txt"},
			{letter: 'R', name: "deleted.txt"},
		}},

		// Error conditions.
		{name: "Dupes", in: "A foo.txt\nM foo.txt\n", err: true, want: []ggStatusLine{
			{letter: 'A', name: "foo.txt"},
		}},
		{name: "Blank", in: "\n", err: true},
		{name: "OneChar", in: "A\n", err: true},
		{name: "NoFile", in: "A \n", err: true},
		{name: "NoSpace", in: "ABfoo.txt\n", err: true},
		{name: "StartSpaced", in: "  foo.txt\n", err: true, want: nil},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			e := new(recordErrorer)
			got := parseGGStatus([]byte(test.in), e)
			diff := cmp.Diff(test.want, got,
				cmp.AllowUnexported(ggStatusLine{}),
				cmpopts.EquateEmpty())
			if diff != "" {
				t.Errorf("parseGGStatus(...) incorrect (-want +got)\n%s", diff)
			}
			if bool(*e) && !test.err {
				t.Error("parseGGStatus(...) reported errors; did not want errors")
			} else if !bool(*e) && test.err {
				t.Error("parseGGStatus(...) did not report errors; want errors")
			}
		})
	}
}

func ggStatusMap(lines []ggStatusLine) map[string]ggStatusLine {
	m := make(map[string]ggStatusLine)
	for _, l := range lines {
		m[l.name] = l
	}
	return m
}

func hasGGStatusLine(lines []ggStatusLine, name string) bool {
	for i := range lines {
		if lines[i].name == name {
			return true
		}
	}
	return false
}

type errorer interface {
	Errorf(format string, args ...interface{})
}

type panicErrorer struct{}

func (panicErrorer) Errorf(format string, args ...interface{}) {
	panic(fmt.Sprintf(format, args...))
}

type recordErrorer bool

func (e *recordErrorer) Errorf(format string, args ...interface{}) {
	*e = true
}
