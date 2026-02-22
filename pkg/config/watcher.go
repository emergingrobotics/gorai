package config

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
)

const debounceDelay = 500 * time.Millisecond

// Watcher monitors a config file for changes and invokes a callback with the
// newly loaded RDL config. It watches the directory containing the file to
// handle editor write-to-temp-then-rename patterns.
type Watcher struct {
	path     string
	fileName string
	onChange func(*RDL)
	logger   *slog.Logger
}

// NewWatcher creates a new config file watcher. The onChange callback is
// invoked with a successfully loaded *RDL after the debounce period.
// Returns an error if the config file path does not exist or is invalid.
func NewWatcher(path string, onChange func(*RDL), logger *slog.Logger) (*Watcher, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("invalid config path: %w", err)
	}
	// Resolve symlinks to prevent redirection attacks
	absPath, err = filepath.EvalSymlinks(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve config path: %w", err)
	}

	return &Watcher{
		path:     absPath,
		fileName: filepath.Base(absPath),
		onChange: onChange,
		logger:   logger,
	}, nil
}

// Start begins watching the config file for changes. It blocks until the
// context is cancelled. The watcher monitors the parent directory to handle
// editor rename patterns.
func (w *Watcher) Start(ctx context.Context) error {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create fsnotify watcher: %w", err)
	}

	dir := filepath.Dir(w.path)
	if err := fsw.Add(dir); err != nil {
		fsw.Close()
		return fmt.Errorf("failed to watch directory %s: %w", dir, err)
	}

	w.logger.Info("Config watcher started", "path", w.path, "directory", dir)

	go w.run(ctx, fsw)
	return nil
}

// run is the main event loop for the watcher.
func (w *Watcher) run(ctx context.Context, fsw *fsnotify.Watcher) {
	defer fsw.Close()

	var debounceTimer *time.Timer

	for {
		select {
		case <-ctx.Done():
			if debounceTimer != nil {
				debounceTimer.Stop()
			}
			w.logger.Debug("Config watcher stopped")
			return

		case event, ok := <-fsw.Events:
			if !ok {
				return
			}
			if filepath.Base(event.Name) != w.fileName {
				continue
			}
			if !event.Has(fsnotify.Write) && !event.Has(fsnotify.Create) && !event.Has(fsnotify.Rename) {
				continue
			}

			// Reset debounce timer
			if debounceTimer != nil {
				debounceTimer.Stop()
			}
			debounceTimer = time.AfterFunc(debounceDelay, func() {
				w.loadAndNotify(ctx)
			})

		case err, ok := <-fsw.Errors:
			if !ok {
				return
			}
			w.logger.Error("Config watcher error", "error", err)
		}
	}
}

// loadAndNotify loads the config file and invokes the onChange callback.
func (w *Watcher) loadAndNotify(ctx context.Context) {
	select {
	case <-ctx.Done():
		return
	default:
	}

	cfg, err := Load(w.path)
	if err != nil {
		w.logger.Error("Config reload: failed to load config", "path", w.path, "error", err)
		return
	}

	if err := cfg.Validate(); err != nil {
		w.logger.Error("Config reload: validation failed", "path", w.path, "error", err)
		return
	}

	w.logger.Info("Config reload: new config loaded successfully", "path", w.path)
	w.onChange(cfg)
}
