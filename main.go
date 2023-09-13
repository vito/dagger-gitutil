package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"strings"
)

type Gitutil struct{}

func (m *Gitutil) DefaultBranch(
	ctx context.Context,
	git *Container,
	repo string,
) (string, error) {
	output, err := git.
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
