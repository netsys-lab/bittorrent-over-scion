package pathselection

import (
	"fmt"
	"sort"

	"github.com/scionproto/scion/go/lib/snet"
)

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

type PathSelectionStore struct {
	data map[string]PeerPathEntry
}

func NewPathSelectionStore() *PathSelectionStore {
	return &PathSelectionStore{
		data: make(map[string]PeerPathEntry, 0),
	}
}

// var PathSelectionStore map[string]PeerPathEntry

// func Init() {
//	PathSelectionStore = make(map[string]PeerPathEntry, 0)
//}

func pathsConflict(path1, path2 snet.Path) bool {
	for _, intP1 := range path1.Metadata().Interfaces {
		for _, intP2 := range path2.Metadata().Interfaces {
			fmt.Printf("Comparing %s:%d to %s:%d resulting in %t\n", intP1.IA, intP1.ID, intP2.IA, intP2.ID, intP1.IA.Equal(intP2.IA) && intP1.ID == intP2.ID)
			if intP1.IA.Equal(intP2.IA) && intP1.ID == intP2.ID {
				return true
			}
		}
	}
	return false
}

// Returns the pathIndex of the first conflicting path, or -1 if no conflicts
func getPeerConflictPaths(path snet.Path, peer PeerPathEntry) int {
	for i, targetPath := range peer.UsedPaths {
		if pathsConflict(path, targetPath) {
			return i
		}
	}

	return -1
}

func getConflictFreePaths(peer PeerPathEntry) []snet.Path {
	paths := make([]snet.Path, 0)
	for _, p1 := range peer.UsedPaths {
		for _, p2 := range paths {
			if !pathsConflict(p1, p2) {
				paths = append(paths, p1)
			}
		}
	}
	return paths
}

// Sorts descending by the number of paths used
func sortPeerPathEntries(entries []PeerPathEntry) []PeerPathEntry {
	// TODO: Avoid in order sorting
	sort.Slice(entries, func(i, j int) bool {
		return len(entries[i].UsedPaths) < len(entries[j].UsedPaths)
	})

	return entries
}

func (p *PathSelectionStore) updatePeerEntryInStore(entry PeerPathEntry) {
	p.data[entry.PeerAddrStr] = entry
}

// TODO: Check the return value, maybe use pointer here...
func removePathFromEntry(entry PeerPathEntry, pathIndex int) PeerPathEntry {
	entry.UsedPaths = append(entry.UsedPaths[:pathIndex], entry.UsedPaths[pathIndex+1:]...)
	return entry
}

// Used paths should be empty here...
func (p *PathSelectionStore) AddPeerEntry(entry PeerPathEntry) {
	potentialConflictingPeers := make([]PeerPathEntry, len(p.data))
	for _, v := range p.data {
		potentialConflictingPeers = append(potentialConflictingPeers, v)
	}
	potentialConflictingPeers = sortPeerPathEntries(potentialConflictingPeers)
	for _, path := range entry.AvailablePaths {

		if len(potentialConflictingPeers) == 0 {
			entry.UsedPaths = append(entry.UsedPaths, path)
			continue
		}
		conflictFound := false
		for _, targetEntry := range potentialConflictingPeers {
			conflictingPathIndex := getPeerConflictPaths(path, targetEntry)
			if conflictingPathIndex >= 0 {
				fmt.Printf("Removing index %d from len %d\n", conflictingPathIndex, len(potentialConflictingPeers))
				// Remove path from targetEntry
				targetEntry = removePathFromEntry(targetEntry, conflictingPathIndex)

				// Replace targetEntry in map
				p.updatePeerEntryInStore(targetEntry)

				// Add to our entry
				entry.UsedPaths = append(entry.UsedPaths, path)
				conflictFound = true
				break
			}
		}

		if !conflictFound {
			entry.UsedPaths = append(entry.UsedPaths, path)
		} else {
			// Update the list, so that we do not steal paths always from the first peer
			potentialConflictingPeers = p.filterByMinimumUsedPaths(potentialConflictingPeers, len(entry.UsedPaths))
		}
	}

	// Filter self containing paths for conflicts
	entry.UsedPaths = getConflictFreePaths(entry)
	p.data[entry.PeerAddrStr] = entry

}

func (p *PathSelectionStore) filterByMinimumUsedPaths(entries []PeerPathEntry, minUsedPath int) []PeerPathEntry {
	newEntries := make([]PeerPathEntry, len(p.data))
	return newEntries
}

// ---------------------------- Deprecated ---------------------------------------

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

// Calculates all potential conflicting (not disjoint) paths
/*
func GetConflictingPaths(entry PeerPathEntry) []ConflictingPathResult {

	return []ConflictingPathResult{}
}

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
*/
