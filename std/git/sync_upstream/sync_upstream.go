package syncupstream

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/cofunclabs/cofunc/functiondriver/go/spec"
	"github.com/cofunclabs/cofunc/manifest"
	"github.com/cofunclabs/cofunc/pkg/output"
)

var branchArg = manifest.UsageDesc{
	Name: "branch",
	Desc: "Specify branches to sync, multiple branches are separated by ',', default main and master",
}

var upstreamArg = manifest.UsageDesc{
	Name: "upstream",
	Desc: "Specify upstream to sync, it not set, will try to find out it from 'git remote -v'",
}

var _manifest = manifest.Manifest{
	Category:    "git",
	Name:        "git_sync_upstream",
	Description: "Sync git branch from upstream",
	Driver:      "go",
	Args: map[string]string{
		branchArg.Name: "main,master",
	},
	RetryOnFailure: 0,
	Usage: manifest.Usage{
		Args:         []manifest.UsageDesc{branchArg},
		ReturnValues: []manifest.UsageDesc{},
	},
}

func New() (*manifest.Manifest, spec.EntrypointFunc, spec.CreateCustomFunc) {
	return &_manifest, Entrypoint, nil
}

func Entrypoint(ctx context.Context, bundle spec.EntrypointBundle, args spec.EntrypointArgs) (map[string]string, error) {
	branches := args.GetStringSlice(branchArg.Name)
	currentBranch, err := getCurrentBranch(ctx)
	if err != nil {
		return nil, err
	}
	var found bool
	for _, branch := range branches {
		if branch == currentBranch {
			found = true
			break
		}
	}
	if !found {
		return map[string]string{"outcome": "no sync: not sync this branch"}, nil
	}

	upstream, err := getUpstreamAddr(ctx)
	if err != nil {
		return nil, err
	}
	if upstream == "" {
		return map[string]string{"outcome": "no sync: not found upstream"}, nil
	}
	// git fetch --all
	if err := fetchRemotes(ctx); err != nil {
		return nil, err
	}
	// git merge-base
	// git merge-tree
	state1, err := checkUpstreamDiff(ctx, currentBranch)
	if err != nil {
		return nil, err
	}
	state2, err := checkOriginDiff(ctx, currentBranch)
	if err != nil {
		return nil, err
	}
	switch state1 {
	case consistent:
		if state2 == consistent {
			return map[string]string{"outcome": "no sync: three branches are consistent"}, nil
		} else {
			if err := pushOrigin(ctx, currentBranch); err != nil {
				return nil, err
			}
		}
		return nil, nil
	case conflict:
		return map[string]string{"outcome": "no sync: branches are conflict"}, nil
	case noConflict:
	}

	// git rebase upstream/branch
	if err := rebaseUpstream(ctx, currentBranch); err != nil {
		return nil, err
	}

	// git push origin branch
	if err := pushOrigin(ctx, currentBranch); err != nil {
		return nil, err
	}

	return map[string]string{"outcome": "synced"}, nil
}

func pushOrigin(ctx context.Context, branch string) error {
	var lasterr error
	for i := 0; i < 3; i++ {
		cmd := exec.CommandContext(ctx, "git", "push", "origin", branch)
		out, err := cmd.CombinedOutput()
		if err != nil {
			lasterr = fmt.Errorf("%w: %s", err, string(out))
			continue
		}
		return nil
	}
	return lasterr
}

func rebaseUpstream(ctx context.Context, branch string) error {
	cmd := exec.CommandContext(ctx, "git", "rebase", "upstream/"+branch)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, string(out))
	}
	return nil
}

type mergeBaseState int

const (
	unknow mergeBaseState = iota
	consistent
	conflict
	noConflict
)

func checkUpstreamDiff(ctx context.Context, branch string) (mergeBaseState, error) {
	return checkMergeBase(ctx, branch, "upstream/"+branch)
}

func checkOriginDiff(ctx context.Context, branch string) (mergeBaseState, error) {
	return checkMergeBase(ctx, "origin/"+branch, branch)
}

func checkMergeBase(ctx context.Context, toBranch, fromBranch string) (mergeBaseState, error) {
	var commitId string
	{
		cmd := exec.CommandContext(ctx, "git", "merge-base", toBranch, fromBranch)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return unknow, fmt.Errorf("%w: %s", err, string(out))
		}
		commitId = strings.TrimSpace(string(out))
	}
	{
		cmd := exec.CommandContext(ctx, "git", "merge-tree", commitId, toBranch, fromBranch)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return unknow, fmt.Errorf("%w: %s", err, string(out))
		}
		if len(out) == 0 {
			return consistent, nil
		}
		if bytes.Contains(out, []byte("\nchanged in both")) {
			return conflict, nil
		}
	}
	return noConflict, nil
}

func fetchRemotes(ctx context.Context) error {
	var lasterr error
	for i := 0; i < 3; i++ {
		cmd := exec.CommandContext(ctx, "git", "fetch", "--all")
		out, err := cmd.CombinedOutput()
		if err != nil {
			lasterr = fmt.Errorf("%w: %s", err, string(out))
			continue
		}
		return nil
	}
	return lasterr
}

func getCurrentBranch(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "branch", "--show-current")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%w: %s", err, string(out))
	}
	return strings.TrimSpace(string(out)), nil
}

func getUpstreamAddr(ctx context.Context) (string, error) {
	var (
		rows [][]string
		sep  = " "
	)
	out := &output.Output{
		W: nil,
		HandleFunc: output.ColumnFunc(&rows, sep, func(fields []string) bool {
			return fields[0] == "upstream" && strings.Contains(fields[2], "fetch")
		}, 0, 1, 2),
	}

	cmd := exec.CommandContext(ctx, "git", "remote", "-v")
	cmd.Stderr = out
	cmd.Stdout = out
	if err := cmd.Run(); err != nil {
		return "", err
	}
	if len(rows) != 0 {
		return rows[0][1], nil
	} else {
		return "", nil
	}
}
