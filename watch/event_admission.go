package watch

import "path/filepath"

type AdmissionResult struct {
	Path     string
	Admitted bool
	Reason   string
}

func AdmitEvent(event Event, cwd string, ignorePatterns []string) AdmissionResult {
	if event.PathType == "watcher" {
		return AdmissionResult{Admitted: false, Reason: "watcher"}
	}
	if cwd == "" {
		return AdmissionResult{Admitted: false, Reason: "relative_path"}
	}

	path, err := filepath.Rel(cwd, event.PathName)
	if err != nil {
		return AdmissionResult{Admitted: false, Reason: "relative_path"}
	}

	if IsIgnored(path, ignorePatterns) {
		return AdmissionResult{Path: path, Admitted: false, Reason: "ignored"}
	}

	if event.EffectType != "modify" && event.EffectType != "create" {
		return AdmissionResult{Path: path, Admitted: false, Reason: "effect"}
	}

	return AdmissionResult{Path: path, Admitted: true}
}
