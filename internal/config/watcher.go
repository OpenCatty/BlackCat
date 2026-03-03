package config

import (
	"os"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Watcher watches a configuration file for changes and calls a callback
// when the file is modified, with debouncing to avoid multiple rapid reloads.
type Watcher struct {
	filePath  string
	onChange  func(*Config)
	watcher   *fsnotify.Watcher
	done      chan struct{}
	mu        sync.RWMutex
	debounce  time.Duration
	lastEvent time.Time
}

// Watch starts watching the configuration file at the provided path.
// When the file changes, it reloads the config and calls the onChange callback
// with the new configuration. The callback is called with debouncing (100ms).
//
// The watch runs in a background goroutine. Call Stop() to clean up.
func Watch(path string, onChange func(*Config)) (*Watcher, error) {
	// Expand ~ to home directory
	if path[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		path = os.ExpandEnv(home + path[1:])
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	w := &Watcher{
		filePath: path,
		onChange: onChange,
		watcher:  watcher,
		done:     make(chan struct{}),
		debounce: 100 * time.Millisecond,
	}

	// Watch the directory containing the config file
	dir := path
	if path != "" {
		// Get directory of the file
		i := len(path) - 1
		for i >= 0 && path[i] != '/' && path[i] != '\\' {
			i--
		}
		if i >= 0 {
			dir = path[:i]
		}
	}

	if err := watcher.Add(dir); err != nil {
		watcher.Close()
		return nil, err
	}

	// Start watching in background
	go w.watchLoop()

	return w, nil
}

// watchLoop monitors fsnotify events and reloads config on changes.
func (w *Watcher) watchLoop() {
	debounceTimer := time.NewTimer(0)
	<-debounceTimer.C // drain initial fire

	for {
		select {
		case <-w.done:
			w.watcher.Close()
			return

		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}

			// Check if this is our config file
			if event.Name != w.filePath {
				continue
			}

			// Only care about writes and creates
			if event.Op&(fsnotify.Write|fsnotify.Create) == 0 {
				continue
			}

			// Reset debounce timer
			debounceTimer.Reset(w.debounce)

		case <-debounceTimer.C:
			// Debounce timeout fired, reload config
			cfg, err := Load(w.filePath)
			if err != nil {
				// Log error but don't stop watching
				continue
			}

			w.mu.RLock()
			onChange := w.onChange
			w.mu.RUnlock()

			if onChange != nil {
				onChange(cfg)
			}

		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			// Log error but continue watching
			_ = err
		}
	}
}

// Stop closes the watcher and stops monitoring for changes.
func (w *Watcher) Stop() error {
	select {
	case <-w.done:
		// Already stopped
		return nil
	default:
		close(w.done)
	}
	return nil
}
