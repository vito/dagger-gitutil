package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"strings"

	"golang.org/x/mod/semver"
)

// GitUtil provides various utilities for working with Git repositories.
type GitUtil struct {
	CustomBase *Container `json:"customBase,omitempty"`
}

// WithRepo sets the repo for future calls to run against.
func (m *GitUtil) Repo(url string) *GitRepo {
	return &GitRepo{
		CustomBase: m.CustomBase,
		URL:        url,
	}
}

// Base returns the base image used for git commands.
func (m *GitRepo) Base() *Container {
	if m.CustomBase != nil {
		return m.CustomBase
	}

	return dag.Apko().Wolfi([]string{"git"})
}

// WithRepo sets the repo for future calls to run against.
func (m *GitUtil) WithBase(base *Container) *GitUtil {
	m.CustomBase = base
	return m
}

// GitRepo represents a Git repository.
type GitRepo struct {
	CustomBase *Container `json:"customBase,omitempty"`
	URL        string     `json:"url"`
}

// DefaultBranch returns the default branch of a git repository.
func (repo *GitRepo) DefaultBranch(ctx context.Context) (string, error) {
	// TODO: this should not be cached. in practice it doesn't change often, but
	// someone setting up a repo for the first time (i.e. a freshly baked Module)
	// might change it soon after indexing.

	output, err := repo.Base().
		WithExec([]string{"git", "ls-remote", "--symref", repo.URL, "HEAD"}, ContainerWithExecOpts{
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

// LatestSemverTag returns the latest semver tag of a git repository.
//
// It accepts an optional prefix which can be used to filter tags. For example,
// if you have multiple sub-directories in a monorepo you might have a
// convention like sub/path/v1.2.3.
func (repo *GitRepo) LatestSemverTag(ctx context.Context, opts struct {
	Prefix string `doc:"Prefix to filter tags by."`
}) (string, error) {
	// TODO: this really should not be cached

	output, err := repo.Base().
		WithExec([]string{"git", "ls-remote", "--tags", repo.URL, opts.Prefix + "v*"}, ContainerWithExecOpts{
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

		refPrefix := "refs/tags/" + opts.Prefix
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
