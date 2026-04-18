package team

import (
	"context"
	"fmt"
	"strings"

	execpkg "github.com/JuliusBrussee/cavekit/internal/exec"
)

func ReadIdentity(root string) (Identity, error) {
	var identity Identity
	if err := readJSON(IdentityPath(root), &identity); err != nil {
		return Identity{}, err
	}
	identity.Email = normalizeEmail(identity.Email)
	return identity, nil
}

func WriteIdentity(root string, identity Identity) error {
	identity.Email = normalizeEmail(identity.Email)
	return writeJSON(IdentityPath(root), identity)
}

func ResolveIdentity(ctx context.Context, executor execpkg.Executor, root, emailFlag, nameFlag string) (Identity, error) {
	email := normalizeEmail(emailFlag)
	if email == "" {
		email = normalizeEmail(gitConfig(ctx, executor, root, "user.email"))
	}
	if email == "" {
		email = normalizeEmail(gitConfig(ctx, executor, root, "--global", "user.email"))
	}
	if email == "" {
		return Identity{}, &ExitError{
			Code:    2,
			Message: "identity resolution failed: set git config user.email or pass --email",
		}
	}

	name := strings.TrimSpace(nameFlag)
	if name == "" {
		name = strings.TrimSpace(gitConfig(ctx, executor, root, "user.name"))
	}
	if name == "" {
		name = strings.TrimSpace(gitConfig(ctx, executor, root, "--global", "user.name"))
	}

	sessionID, err := newUUID()
	if err != nil {
		return Identity{}, fmt.Errorf("generate session id: %w", err)
	}

	return Identity{
		Email:    email,
		Name:     name,
		Session:  sessionID,
		JoinedAt: nowRFC3339(),
	}, nil
}

func gitConfig(ctx context.Context, executor execpkg.Executor, root string, args ...string) string {
	gitArgs := append([]string{"config"}, args...)
	res, err := executor.RunDir(ctx, root, "git", gitArgs...)
	if err != nil || res.ExitCode != 0 {
		return ""
	}
	return strings.TrimSpace(res.Stdout)
}
