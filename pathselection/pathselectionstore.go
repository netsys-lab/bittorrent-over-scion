package pathselection

import "github.com/scionproto/scion/go/lib/snet"

type PeerPathEntry struct {
	PeerAddrStr    string
	PeerAddr       snet.UDPAddr
	AvailablePaths []snet.Path
	UsedPaths      []snet.Path
}

type ConflictingPathResult struct {
	PeerAddrStr       string
	PeerAddr          snet.UDPAddr
	ConflictingPaths  []snet.Path
	NumPathsInUse     int
	NumPathsAvailable int
}

var PathSelectionStore map[string]PeerPathEntry

func Init() {
	PathSelectionStore = make(map[string]PeerPathEntry, 0)
}

// Calculates all potential conflicting (not disjoint) paths
func GetConflictingPaths(entry PeerPathEntry) []ConflictingPathResult {
	return []ConflictingPathResult{}
}

// How to use
// We have a new peer with a set of (used and) available paths
// We call GetConflictingPaths(peer) to get all other peers and potential conflicting paths
// How to determine which peers to "steal" paths:
// We sort the result of GetConflictingPaths based on NumPathsInUse. Afterwards, we filter live
// So that we only have peers that have #usedPath > #curUsedPath of our peer
// For each of those peers, we "steal" one path and add a path to our peer
// We support stealing multiple paths by iterating over the list multiple times
// TODO: Conflicts are not relevant for all paths, but potentially one path per peer?

// Beginning Assumptions:
// New peer has only 1 path
// Current Peer has multiple paths

func sortConflicts([]ConflictingPathResult) {

}

func DoStuff(entry PeerPathEntry) {
	conflicts := GetConflictingPaths(entry)

	sortConflicts(conflicts)

	for len(conflicts) > 0 {
		pathsGot := false
		for i := 0; i < len(conflicts); i++ {
			if conflicts[i].NumPathsInUse > len(entry.UsedPaths) {
				// TODO: Steal path here
				conflicts[i].NumPathsInUse--
				pathsGot = true
			}
		}

		if !pathsGot {
			break
		}

	}
}
