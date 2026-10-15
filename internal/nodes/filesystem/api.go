// Copyright (C) 2026 Dennis Bleul
// Licensed under the GNU Affero General Public License v3.0 or later.
// See LICENSE file for details.

// This file exposes the small public surface of the filesystem package:
// the files folder (set at startup) and name resolution inside it, used by
// the files API of the editor (Settings → Files).

package filesystem

import "sync"

var (
	workdirMu sync.RWMutex
	workdir   string
)

// SetWorkdir installs the mandatory filesystem jail every File/CSV node is
// confined to from then on, regardless of any rootJail a flow configures —
// OrbitOS apps may only touch their own workdir. main() calls this once at
// startup with the app's data directory, after ensuring it exists (the jail
// check requires the directory to already be there). An empty path (the
// zero value — tests, or loopze-edge run outside OrbitOS) disables the
// mandatory floor; see effectiveJail in helpers.go.
func SetWorkdir(path string) {
	workdirMu.Lock()
	workdir = path
	workdirMu.Unlock()
}

func mandatoryWorkdir() string {
	workdirMu.RLock()
	defer workdirMu.RUnlock()
	return workdir
}

// Workdir is the folder the file nodes are confined to (the app's files
// folder on Orbit OS), or "" when none was set.
func Workdir() string {
	return mandatoryWorkdir()
}

// Resolve turns a name inside the files folder ("logs/readings.csv") into
// an absolute path, refusing anything outside the folder.
func Resolve(name string) (string, error) {
	return validatePath(name, "")
}

// RelName is the name of an absolute path inside the files folder, as the
// user sees it ("logs/readings.csv"); other paths are returned unchanged.
func RelName(abs string) string {
	return relName(abs)
}
