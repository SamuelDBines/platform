package nunchucks

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type PrecompileOptions struct {
	OutputFormat string
	// Ignore is a list of glob patterns (slash paths) relative to the views root.
	// Matched paths are skipped (no render, no static copy). See TemplateRelIgnored.
	Ignore []string
}

func IsTemplateFile(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".njk", ".html", ".txt", ".yaml", ".yml", ".json", ".xml", ".css", ".js":
		return true
	default:
		return false
	}
}

func outputPathFor(rel string, opts PrecompileOptions) string {
	if strings.EqualFold(strings.TrimSpace(opts.OutputFormat), "html") && strings.EqualFold(filepath.Ext(rel), ".njk") {
		return strings.TrimSuffix(rel, filepath.Ext(rel)) + ".html"
	}
	return rel
}

func copyStaticFile(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	mode := info.Mode() & 0o777
	if mode == 0 {
		mode = 0o644
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

// PrecompileDir renders templates from the configured views path into outDir.
func (e *Env) PrecompileDir(outDir string, ctx map[string]any) error {
	return e.PrecompileDirWithOptions(outDir, ctx, PrecompileOptions{})
}

// PrecompileDirWithOptions renders templates from the configured views path into outDir.
func (e *Env) PrecompileDirWithOptions(outDir string, ctx map[string]any, opts PrecompileOptions) error {
	if strings.TrimSpace(outDir) == "" {
		return fmt.Errorf("outDir is required")
	}
	if strings.TrimSpace(e.basePath) == "" {
		return fmt.Errorf("env base path is empty")
	}
	if format := strings.TrimSpace(opts.OutputFormat); format != "" && !strings.EqualFold(format, "preserve") && !strings.EqualFold(format, "html") {
		return fmt.Errorf("unsupported output format: %s", format)
	}

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}

	return filepath.WalkDir(e.basePath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		rel, err := filepath.Rel(e.basePath, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if TemplateRelIgnored(rel, opts.Ignore) {
			return nil
		}

		if !IsTemplateFile(d.Name()) {
			dst := filepath.Join(outDir, filepath.FromSlash(rel))
			if err := copyStaticFile(path, dst); err != nil {
				return fmt.Errorf("copy %s: %w", rel, err)
			}
			return nil
		}

		rendered, err := e.Render(rel, ctx)
		if err != nil {
			return fmt.Errorf("render %s: %w", rel, err)
		}

		dst := filepath.Join(outDir, filepath.FromSlash(outputPathFor(rel, opts)))
		if strings.TrimSpace(rendered) == "" {
			_ = os.Remove(dst)
			return nil
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		return os.WriteFile(dst, []byte(rendered), 0o644)
	})
}
