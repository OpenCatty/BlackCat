package cmd

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/startower-observability/blackcat/internal/config"
	"github.com/startower-observability/blackcat/internal/llm"
	"github.com/startower-observability/blackcat/internal/llm/copilot"
	"github.com/startower-observability/blackcat/internal/llm/gemini"
	"github.com/startower-observability/blackcat/internal/llm/zen"
	"github.com/startower-observability/blackcat/internal/types"
)

const (
	initDeepDefaultMaxDepth = 3
	initDeepRateLimitDelay  = 100 * time.Millisecond
)

var (
	initDeepCmd = &cobra.Command{
		Use:   "init-deep [directory]",
		Short: "Generate hierarchical AGENTS.md files using the configured LLM",
		Long: `init-deep walks a directory tree and generates AGENTS.md files
for each eligible directory using the configured LLM client.`,
		Args: cobra.MaximumNArgs(1),
		RunE: runInitDeep,
	}

	initDeepNewLLMClient = initDeepLLMFromConfig
	initDeepSleep        = time.Sleep
)

type initDeepTarget struct {
	Dir        string
	AgentsPath string
	Files      []string
}

func init() {
	rootCmd.AddCommand(initDeepCmd)
	initDeepCmd.Flags().Bool("dry-run", false, "Show generated AGENTS.md content without writing files")
	initDeepCmd.Flags().Bool("force", false, "Overwrite existing AGENTS.md files")
	initDeepCmd.Flags().Int("max-depth", initDeepDefaultMaxDepth, "Maximum directory depth to process")
}

func runInitDeep(cmd *cobra.Command, args []string) error {
	rootDir := "."
	if len(args) > 0 {
		rootDir = args[0]
	}

	dryRun, err := cmd.Flags().GetBool("dry-run")
	if err != nil {
		return fmt.Errorf("read --dry-run flag: %w", err)
	}
	force, err := cmd.Flags().GetBool("force")
	if err != nil {
		return fmt.Errorf("read --force flag: %w", err)
	}
	maxDepth, err := cmd.Flags().GetInt("max-depth")
	if err != nil {
		return fmt.Errorf("read --max-depth flag: %w", err)
	}
	if maxDepth < 0 {
		return fmt.Errorf("--max-depth must be >= 0")
	}

	absRoot, err := filepath.Abs(rootDir)
	if err != nil {
		return fmt.Errorf("resolve root directory: %w", err)
	}
	absRoot = filepath.Clean(absRoot)

	slog.Info("init-deep scanning workspace", "root", absRoot, "max_depth", maxDepth, "dry_run", dryRun, "force", force)
	targets, err := collectInitDeepTargets(absRoot, force, maxDepth)
	if err != nil {
		return err
	}
	if len(targets) == 0 {
		slog.Info("init-deep found no eligible directories", "root", absRoot)
		return nil
	}

	llmClient, err := initDeepNewLLMClient()
	if err != nil {
		return fmt.Errorf("init-deep LLM client: %w", err)
	}

	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	for i, target := range targets {
		prompt := buildInitDeepPrompt(target.Dir, target.Files)
		resp, chatErr := llmClient.Chat(ctx, []types.LLMMessage{{Role: "user", Content: prompt}}, nil)
		if chatErr != nil {
			return fmt.Errorf("generate AGENTS.md for %s: %w", target.Dir, chatErr)
		}

		content := strings.TrimSpace(resp.Content)
		if content == "" {
			return fmt.Errorf("generate AGENTS.md for %s: empty response", target.Dir)
		}

		if dryRun {
			slog.Info("init-deep dry run generated AGENTS.md", "path", target.AgentsPath)
			cmd.Printf("[dry-run] %s\n%s\n\n", target.AgentsPath, content)
		} else {
			if !strings.HasSuffix(content, "\n") {
				content += "\n"
			}
			if writeErr := os.WriteFile(target.AgentsPath, []byte(content), 0o644); writeErr != nil {
				return fmt.Errorf("write %s: %w", target.AgentsPath, writeErr)
			}
			slog.Info("init-deep wrote AGENTS.md", "path", target.AgentsPath)
		}

		if i < len(targets)-1 {
			initDeepSleep(initDeepRateLimitDelay)
		}
	}

	slog.Info("init-deep completed", "generated", len(targets), "dry_run", dryRun)
	return nil
}

func collectInitDeepTargets(rootDir string, force bool, maxDepth int) ([]initDeepTarget, error) {
	dirFiles := make(map[string][]string)

	err := filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		if d.IsDir() {
			depth, depthErr := initDeepDepth(rootDir, path)
			if depthErr != nil {
				return depthErr
			}

			if depth > 0 && strings.HasPrefix(d.Name(), ".") {
				slog.Debug("init-deep skipping hidden directory", "path", path)
				return filepath.SkipDir
			}
			if depth > maxDepth {
				slog.Debug("init-deep skipping depth-capped directory", "path", path, "depth", depth, "max_depth", maxDepth)
				return filepath.SkipDir
			}
			return nil
		}

		dirDepth, depthErr := initDeepDepth(rootDir, filepath.Dir(path))
		if depthErr != nil {
			return depthErr
		}

		if dirDepth > maxDepth {
			return nil
		}

		name := d.Name()
		if !isInitDeepSourceFile(name) {
			return nil
		}

		dir := filepath.Dir(path)
		dirFiles[dir] = append(dirFiles[dir], name)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk root %s: %w", rootDir, err)
	}

	dirs := make([]string, 0, len(dirFiles))
	for dir := range dirFiles {
		dirs = append(dirs, dir)
	}
	sort.Strings(dirs)

	targets := make([]initDeepTarget, 0, len(dirs))
	for _, dir := range dirs {
		files := dirFiles[dir]
		if len(files) == 0 {
			continue
		}
		sort.Strings(files)

		agentsPath := filepath.Join(dir, "AGENTS.md")
		if !force {
			_, statErr := os.Stat(agentsPath)
			if statErr == nil {
				slog.Info("init-deep skipping existing AGENTS.md", "path", agentsPath)
				continue
			}
			if !os.IsNotExist(statErr) {
				return nil, fmt.Errorf("check %s: %w", agentsPath, statErr)
			}
		}

		targets = append(targets, initDeepTarget{
			Dir:        dir,
			AgentsPath: agentsPath,
			Files:      files,
		})
	}

	return targets, nil
}

func buildInitDeepPrompt(dir string, files []string) string {
	return fmt.Sprintf(
		"You are generating an AGENTS.md file for the directory `%s`. The directory contains these files: %s. Generate a concise AGENTS.md (max 200 words) describing the directory's purpose and key patterns.",
		dir,
		strings.Join(files, ", "),
	)
}

func initDeepDepth(rootDir, path string) (int, error) {
	rel, err := filepath.Rel(rootDir, path)
	if err != nil {
		return 0, fmt.Errorf("relative path from %s to %s: %w", rootDir, path, err)
	}
	if rel == "." {
		return 0, nil
	}

	rel = filepath.ToSlash(filepath.Clean(rel))
	return strings.Count(rel, "/") + 1, nil
}

func isInitDeepSourceFile(name string) bool {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".go", ".md", ".yaml", ".json":
		return true
	default:
		return false
	}
}

func initDeepLLMFromConfig() (types.LLMClient, error) {
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	llm.RegisterBackend("openai", llm.NewOpenAIBackend)
	llm.RegisterBackend("copilot", func(bc llm.BackendConfig) (llm.Backend, error) {
		return copilot.NewCopilotBackend(bc)
	})
	llm.RegisterBackend("gemini", func(bc llm.BackendConfig) (llm.Backend, error) {
		return gemini.NewGeminiBackend(bc)
	})
	llm.RegisterBackend("zen", func(bc llm.BackendConfig) (llm.Backend, error) {
		return zen.NewZenBackend(bc)
	})

	if activeBackend, provider := createActiveBackend(cfg); activeBackend != nil {
		slog.Info("init-deep using phase 2 backend", "provider", provider)
		return backendAdapter{backend: activeBackend}, nil
	}

	return llm.NewClient(
		cfg.LLM.APIKey,
		cfg.LLM.BaseURL,
		cfg.LLM.Model,
		cfg.LLM.Temperature,
		cfg.LLM.MaxTokens,
	), nil
}
