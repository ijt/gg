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
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"zombiezen.com/go/gut/internal/flag"
	"zombiezen.com/go/gut/internal/gittool"
)

const commitSynopsis = "commit the specified files or all outstanding changes"

func commit(ctx context.Context, git *gittool.Tool, args []string) error {
	f := flag.NewFlagSet(true, "gut commit [--amend] [-m MSG] [FILE [...]]", commitSynopsis)
	amend := f.Bool("amend", false, "amend the parent of the working directory")
	msg := f.String("m", "", "use text as commit `message`")
	if err := f.Parse(args); flag.IsHelp(err) {
		f.Help(os.Stdout)
		return nil
	} else if err != nil {
		return usagef("%v", err)
	}
	var commitArgs []string
	commitArgs = append(commitArgs, "commit")
	if *amend {
		commitArgs = append(commitArgs, "--amend")
	}
	if *msg != "" {
		commitArgs = append(commitArgs, "--message="+*msg)
	}
	commitArgs = append(commitArgs, "--")
	if f.NArg() == 0 {
		var err error
		commitArgs, err = inferCommitFiles(ctx, git, commitArgs)
		if err != nil {
			return err
		}
	} else {
		commitArgs = append(commitArgs, f.Args()...)
	}
	return git.RunInteractive(ctx, commitArgs...)
}

func inferCommitFiles(ctx context.Context, git *gittool.Tool, files []string) ([]string, error) {
	missing, missingStaged, unmerged := 0, 0, 0
	p, err := git.Start(ctx, "status", "--porcelain=v1", "-z", "-unormal")
	if err != nil {
		return files, err
	}
	defer p.Wait()
	r := bufio.NewReader(p)
	filesStart := len(files)
	for {
		ent, err := readStatusEntry(r)
		if err == io.EOF {
			break
		}
		if err != nil {
			return files[:filesStart], err
		}
		switch {
		case ent.isMissing():
			missing++
			if ent.code[0] != ' ' {
				missingStaged++
			}
		case ent.isAdded() || ent.isModified() || ent.isRemoved() || ent.isCopied() || ent.isRenamed():
			// Prepend ":/:" pathspec prefix, because status reports from top of repository.
			files = append(files, ":/:"+ent.name)
		case ent.isIgnored() || ent.isUntracked():
			// Skip
		case ent.isUnmerged():
			unmerged++
		default:
			panic("unhandled status code")
		}
	}
	if unmerged == 1 {
		return files[:filesStart], errors.New("1 unmerged file; see 'gut status'")
	}
	if unmerged > 1 {
		return files[:filesStart], fmt.Errorf("%d unmerged files; see 'gut status'", unmerged)
	}
	if len(files) == filesStart {
		switch missing {
		case 0:
			return files[:filesStart], errors.New("nothing changed")
		case 1:
			return files[:filesStart], errors.New("nothing changed (1 missing file; see 'gut status')")
		default:
			return files[:filesStart], fmt.Errorf("nothing changed (%d missing files; see 'gut status')", missing)
		}
	}
	if missingStaged == 1 {
		return files[:filesStart], errors.New("git has staged changes for 1 missing file; see 'gut status'")
	}
	if missingStaged > 1 {
		return files[:filesStart], fmt.Errorf("git has staged changes for %d missing file; see 'gut status'", missingStaged)
	}
	return files, p.Wait()
}