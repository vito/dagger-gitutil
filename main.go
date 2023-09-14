package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"strings"

	"golang.org/x/mod/semver"
)

type Gitutil struct{}

// DefaultBranch returns the default branch of a git repository.
func (m *Gitutil) DefaultBranch(ctx context.Context, gitBase *Container, repo string) (string, error) {
	output, err := gitBase.
		WithExec([]string{"git", "ls-remote", "--symref", repo, "HEAD"}, ContainerWithExecOpts{
			SkipEntrypoint: true,
		}).
		Stdout(ctx)
	if err != nil {
		return "", err
	}

	scanner := bufio.NewScanner(bytes.NewBufferString(output))

	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 3 {
			continue
		}

		if fields[0] == "ref:" && fields[2] == "HEAD" {
			return strings.TrimPrefix(fields[1], "refs/heads/"), nil
		}
	}

	return "", fmt.Errorf("could not deduce default branch from output:\n%s", output)
}

func (m *Gitutil) LatestSemverTag(ctx context.Context, gitBase *Container, repo string, prefix string) (string, error) {
	output, err := gitBase.
		WithExec([]string{"git", "ls-remote", "--tags", repo, prefix + "v*"}, ContainerWithExecOpts{
			SkipEntrypoint: true,
		}).
		Stdout(ctx)
	if err != nil {
		return "", err
	}

	scanner := bufio.NewScanner(bytes.NewBufferString(output))

	var versions []string
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 2 {
			continue
		}

		refPrefix := "refs/tags/" + prefix
		if !strings.HasPrefix(fields[1], refPrefix) {
			continue
		}

		tag := strings.TrimPrefix(fields[1], refPrefix)

		if semver.IsValid(tag) {
			versions = append(versions, tag)
		}
	}

	semver.Sort(versions)

	if len(versions) > 0 {
		return versions[len(versions)-1], nil
	}

	return "", fmt.Errorf("no versions present")
}
