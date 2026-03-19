package runner

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	github "github.com/LegationPro/zagforge-mvp-impl/shared/go/provider/github"
	"go.uber.org/zap"
)

// RepoCloner is the subset of provider.Worker the runner needs.
type RepoCloner interface {
	GenerateCloneToken(ctx context.Context, installationID int64) (string, error)
	CloneRepo(ctx context.Context, repoURL, ref, token, dst string) error
}

// Config holds runner settings, all resolvable from environment variables via config.LoadWorkerConfig.
type Config struct {
	WorkspaceDir string // base dir for temporary clone directories
	ZigzagBin    string // path to the zigzag binary
	ReportsDir   string // absolute path where zigzag writes reports
}

// Runner clones a repo, runs zigzag, then cleans up the temporary clone.
type Runner struct {
	cloner RepoCloner
	cfg    Config
	log    *zap.Logger
	wg     sync.WaitGroup
}

func New(cloner RepoCloner, cfg Config, log *zap.Logger) *Runner {
	return &Runner{cloner: cloner, cfg: cfg, log: log}
}

// Dispatch satisfies handler.Dispatcher. It runs the job in a goroutine,
// detached from the HTTP request context so the handler can return immediately.
func (r *Runner) Dispatch(ctx context.Context, event github.WebhookEvent) {
	r.wg.Go(func() {
		if err := r.Run(context.Background(), event); err != nil {
			r.log.Error("job failed",
				zap.String("repo", event.RepoName),
				zap.String("branch", event.Branch),
				zap.String("commit", event.CommitSHA),
				zap.Error(err),
			)
		}
	})
}

// Wait blocks until all in-flight jobs complete. Call during graceful shutdown.
func (r *Runner) Wait() {
	r.wg.Wait()
}

// Run executes the full job: generate token → clone → zigzag → cleanup.
func (r *Runner) Run(ctx context.Context, event github.WebhookEvent) error {
	r.log.Info("starting job",
		zap.String("repo", event.RepoName),
		zap.String("branch", event.Branch),
		zap.String("commit", event.CommitSHA),
	)

	token, err := r.cloner.GenerateCloneToken(ctx, event.InstallationID)
	if err != nil {
		return fmt.Errorf("generate clone token: %w", err)
	}

	if err := os.MkdirAll(r.cfg.WorkspaceDir, 0o755); err != nil {
		return fmt.Errorf("create workspace dir: %w", err)
	}

	workDir, err := os.MkdirTemp(r.cfg.WorkspaceDir, "job-*")
	if err != nil {
		return fmt.Errorf("create work dir: %w", err)
	}

	defer func(workDir string) {
		if err := os.RemoveAll(workDir); err != nil {
			r.log.Warn("failed to remove work dir", zap.String("path", workDir), zap.Error(err))
		}
	}(workDir)

	repoDir := filepath.Join(workDir, "repo")
	if err := r.cloner.CloneRepo(ctx, event.CloneURL, event.Branch, token, repoDir); err != nil {
		return fmt.Errorf("clone repo: %w", err)
	}

	r.log.Info("running zigzag",
		zap.String("repo", event.RepoName),
		zap.String("reports_dir", r.cfg.ReportsDir),
	)
	cmd := exec.CommandContext(ctx, r.cfg.ZigzagBin, "run", "--no-watch", "--output-dir", r.cfg.ReportsDir)
	cmd.Dir = repoDir
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("zigzag run: %w: %s", err, out)
	}

	r.log.Info("job complete",
		zap.String("repo", event.RepoName),
		zap.String("branch", event.Branch),
		zap.String("commit", event.CommitSHA),
		zap.String("reports_dir", r.cfg.ReportsDir),
	)
	return nil
}
