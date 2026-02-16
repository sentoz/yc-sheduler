package reloader

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

// Reloader watches schedules directory and applies updates on changes.
type Reloader struct {
	onChange     func(context.Context) error
	schedulesDir string
	interval     time.Duration
	lastSig      [sha256.Size]byte
	hasLastSig   bool
}

// New creates a new schedules reloader.
func New(schedulesDir string, interval time.Duration, onChange func(context.Context) error) (*Reloader, error) {
	if schedulesDir == "" {
		return nil, fmt.Errorf("reloader: empty schedules directory")
	}
	if interval <= 0 {
		return nil, fmt.Errorf("reloader: interval must be greater than zero")
	}
	if onChange == nil {
		return nil, fmt.Errorf("reloader: onChange callback is required")
	}

	return &Reloader{
		schedulesDir: schedulesDir,
		interval:     interval,
		onChange:     onChange,
	}, nil
}

// Start begins watching schedules directory until ctx is canceled.
func (r *Reloader) Start(ctx context.Context) {
	if r == nil {
		return
	}

	if sig, err := calcDirSignature(r.schedulesDir); err != nil {
		log.Warn().Err(err).Str("schedules_dir", r.schedulesDir).Msg("Failed to initialize schedules watcher signature")
	} else {
		r.lastSig = sig
		r.hasLastSig = true
	}

	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	log.Info().
		Str("schedules_dir", r.schedulesDir).
		Dur("interval", r.interval).
		Msg("Schedules auto-reload watcher started")

	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("Schedules auto-reload watcher stopped")
			return
		case <-ticker.C:
			r.tick(ctx)
		}
	}
}

func (r *Reloader) tick(ctx context.Context) {
	sig, err := calcDirSignature(r.schedulesDir)
	if err != nil {
		log.Warn().Err(err).Str("schedules_dir", r.schedulesDir).Msg("Failed to read schedules directory state")
		return
	}

	if r.hasLastSig && sig == r.lastSig {
		return
	}

	log.Info().Str("schedules_dir", r.schedulesDir).Msg("Detected schedules change, applying reload")
	if err := r.onChange(ctx); err != nil {
		log.Error().Err(err).Str("schedules_dir", r.schedulesDir).Msg("Schedules reload failed, keeping previous schedule set")
	} else {
		log.Info().Str("schedules_dir", r.schedulesDir).Msg("Schedules reload applied")
	}

	r.lastSig = sig
	r.hasLastSig = true
}

func calcDirSignature(path string) ([sha256.Size]byte, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return [sha256.Size]byte{}, fmt.Errorf("read dir %q: %w", path, err)
	}

	hasher := sha256.New()
	fileNames := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if ext != ".yaml" && ext != ".yml" {
			continue
		}
		fileNames = append(fileNames, entry.Name())
	}

	sort.Strings(fileNames)
	for _, name := range fileNames {
		fullPath := filepath.Join(path, name)
		data, err := os.ReadFile(fullPath)
		if err != nil {
			return [sha256.Size]byte{}, fmt.Errorf("read file %q: %w", fullPath, err)
		}

		if _, err := hasher.Write([]byte(name)); err != nil {
			return [sha256.Size]byte{}, fmt.Errorf("hash filename %q: %w", name, err)
		}
		if _, err := hasher.Write([]byte{0}); err != nil {
			return [sha256.Size]byte{}, fmt.Errorf("hash separator for %q: %w", name, err)
		}
		if _, err := hasher.Write(data); err != nil {
			return [sha256.Size]byte{}, fmt.Errorf("hash file %q: %w", fullPath, err)
		}
		if _, err := hasher.Write([]byte{0}); err != nil {
			return [sha256.Size]byte{}, fmt.Errorf("hash tail separator for %q: %w", name, err)
		}
	}

	var sum [sha256.Size]byte
	copy(sum[:], hasher.Sum(nil))
	return sum, nil
}
