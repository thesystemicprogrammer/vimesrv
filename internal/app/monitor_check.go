package app

import (
	"os"
	"strings"
)

type ScreenState struct {
	HasDisplayServer bool
	HasMonitor       bool
	UsableScreen     bool
}

func DetectScreen() ScreenState {
	hasDisplay := os.Getenv("DISPLAY") != "" || os.Getenv("WAYLAND_DISPLAY") != ""
	hasMonitor := hasConnectedMonitor()

	return ScreenState{
		HasDisplayServer: hasDisplay,
		HasMonitor:       hasMonitor,
		UsableScreen:     hasDisplay && hasMonitor,
	}
}

func hasConnectedMonitor() bool {
	drmPath := "/sys/class/drm"

	entries, err := os.ReadDir(drmPath)
	if err != nil {
		return false
	}

	for _, e := range entries {
		if !strings.Contains(e.Name(), "-") {
			continue
		}

		statusPath := drmPath + "/" + e.Name() + "/status"
		data, err := os.ReadFile(statusPath)
		if err != nil {
			continue
		}

		if strings.TrimSpace(string(data)) == "connected" {
			return true
		}
	}

	return false
}
