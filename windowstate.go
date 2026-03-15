package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"
)

const (
	windowModeNormal     = "normal"
	windowModeMaximised  = "maximised"
	windowModeFullscreen = "fullscreen"
)

type windowBounds struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

func (b windowBounds) valid() bool {
	return b.Width > 0 && b.Height > 0
}

func windowBoundsFromRect(rect application.Rect) windowBounds {
	return windowBounds{
		X:      rect.X,
		Y:      rect.Y,
		Width:  rect.Width,
		Height: rect.Height,
	}
}

type persistedWindowState struct {
	Version      int          `json:"version"`
	Mode         string       `json:"mode"`
	Bounds       windowBounds `json:"bounds"`
	NormalBounds windowBounds `json:"normalBounds"`
}

func (s persistedWindowState) modeOrDefault() string {
	switch s.Mode {
	case windowModeNormal, windowModeMaximised, windowModeFullscreen:
		return s.Mode
	default:
		return windowModeNormal
	}
}

func (s persistedWindowState) startupBounds() (windowBounds, bool) {
	if s.NormalBounds.valid() {
		return s.NormalBounds, true
	}
	if s.Bounds.valid() {
		return s.Bounds, true
	}
	return windowBounds{}, false
}

func (s persistedWindowState) normalized() persistedWindowState {
	s.Mode = s.modeOrDefault()
	if !s.Bounds.valid() {
		s.Bounds = windowBounds{}
	}
	if !s.NormalBounds.valid() {
		s.NormalBounds = windowBounds{}
	}
	return s
}

type windowStateStore struct {
	path string
	mu   sync.Mutex
}

func newWindowStateStore(appName string) (*windowStateStore, error) {
	if appName == "" {
		return nil, errors.New("app name is required")
	}

	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}

	statePath := filepath.Join(configDir, appName, "window-state.json")
	return &windowStateStore{path: statePath}, nil
}

func (s *windowStateStore) Load() (persistedWindowState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return persistedWindowState{
				Version: 1,
				Mode:    windowModeNormal,
			}, nil
		}
		return persistedWindowState{}, err
	}

	var state persistedWindowState
	if err := json.Unmarshal(data, &state); err != nil {
		return persistedWindowState{}, err
	}

	if state.Version == 0 {
		state.Version = 1
	}
	return state.normalized(), nil
}

func (s *windowStateStore) Save(state persistedWindowState) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	state = state.normalized()
	if state.Version == 0 {
		state.Version = 1
	}

	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	payload, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	tempFile, err := os.CreateTemp(dir, "window-state-*.tmp")
	if err != nil {
		return err
	}
	tempPath := tempFile.Name()

	cleanup := func() {
		_ = tempFile.Close()
		_ = os.Remove(tempPath)
	}

	if _, err := tempFile.Write(payload); err != nil {
		cleanup()
		return err
	}
	if err := tempFile.Sync(); err != nil {
		cleanup()
		return err
	}
	if err := tempFile.Close(); err != nil {
		_ = os.Remove(tempPath)
		return err
	}
	if err := os.Rename(tempPath, s.path); err != nil {
		_ = os.Remove(tempPath)
		return err
	}

	return nil
}
