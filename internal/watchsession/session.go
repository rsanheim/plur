package watchsession

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/rsanheim/plur/internal/runtime"
	"github.com/rsanheim/plur/job"
	"github.com/rsanheim/plur/watch"
)

type Options struct {
	IgnorePatterns   []string
	FilterWatchDirs  bool
	RequireWatchDirs bool
}

type Session struct {
	Selected       *runtime.SelectedJob
	Jobs           map[string]job.Job
	Watches        []watch.WatchMapping
	RawWatchDirs   []string
	WatchDirs      []string
	IgnorePatterns []string
	CWD            string
	Planner        watch.Planner
}

func New(rc *runtime.RuntimeConfig, opts Options) (*Session, error) {
	selected, err := runtime.SelectJobFromRuntimeConfig(rc, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to select watch job: %w", err)
	}

	cwd, err := CurrentWorkingDirectory()
	if err != nil {
		return nil, err
	}

	rawWatchDirs := WatchDirectories(rc.Watches)
	watchDirs := slices.Clone(rawWatchDirs)
	if opts.FilterWatchDirs {
		watchDirs, err = watch.FilterDirectories(watchDirs)
		if err != nil {
			return nil, fmt.Errorf("failed to filter watch directories: %w", err)
		}
	}
	if opts.RequireWatchDirs && len(watchDirs) == 0 {
		return nil, fmt.Errorf("no directories to watch found in watch mappings")
	}

	ignorePatterns := slices.Clone(opts.IgnorePatterns)
	if len(ignorePatterns) == 0 {
		ignorePatterns = slices.Clone(watch.DefaultIgnorePatterns)
	}

	session := &Session{
		Selected:       selected,
		Jobs:           rc.Jobs,
		Watches:        rc.Watches,
		RawWatchDirs:   rawWatchDirs,
		WatchDirs:      watchDirs,
		IgnorePatterns: ignorePatterns,
		CWD:            cwd,
	}
	session.Planner = watch.Planner{
		Jobs:    session.Jobs,
		Watches: session.Watches,
		CWD:     session.CWD,
	}
	return session, nil
}

func CurrentWorkingDirectory() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}
	if resolvedCwd, err := filepath.EvalSymlinks(cwd); err == nil {
		cwd = resolvedCwd
	}
	return cwd, nil
}

func WatchDirectories(watches []watch.WatchMapping) []string {
	dirs := make([]string, 0, len(watches))
	for _, mapping := range watches {
		dirs = append(dirs, mapping.SourceDir())
	}
	return dirs
}

func NormalizePath(cwd, filePath string) string {
	if filepath.IsAbs(filePath) {
		if rel, err := filepath.Rel(cwd, filePath); err == nil {
			return rel
		}
	}
	return filePath
}

func (s *Session) NormalizePath(filePath string) string {
	return NormalizePath(s.CWD, filePath)
}

func (s *Session) PlanPath(filePath string) watch.Plan {
	return s.Planner.PlanPath(s.NormalizePath(filePath))
}

func (s *Session) AdmitEvent(event watch.Event) watch.AdmissionResult {
	return watch.AdmitEvent(event, s.CWD, s.IgnorePatterns)
}

func (s *Session) AdmitPathForPreview(filePath string) watch.AdmissionResult {
	pathName := filePath
	if !filepath.IsAbs(pathName) {
		pathName = filepath.Join(s.CWD, pathName)
	}
	return s.AdmitEvent(watch.Event{
		PathType:   "file",
		PathName:   pathName,
		EffectType: "modify",
	})
}

func (s *Session) Handler() *watch.FileEventHandler {
	return &watch.FileEventHandler{
		Jobs:    s.Jobs,
		Watches: s.Watches,
		CWD:     s.CWD,
	}
}
