package pathselection

import (
	"fmt"
	"sort"

	"github.com/scionproto/scion/go/lib/snet"
	log "github.com/sirupsen/logrus"
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

func pathsConflict(path1, path2 snet.Path) bool {
	for _, intP1 := range path1.Metadata().Interfaces {
		for _, intP2 := range path2.Metadata().Interfaces {
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
		pathsConflicted := false
		for _, p2 := range paths {
			if pathsConflict(p1, p2) {
				pathsConflicted = true
				break
			}
		}

		if !pathsConflicted {
			paths = append(paths, p1)
		}
	}
	return paths
}

// Sorts descending by the number of paths used
func sortPeerPathEntries(entries []PeerPathEntry) []PeerPathEntry {
	sort.Slice(entries, func(i, j int) bool {
		return len(entries[i].UsedPaths) < len(entries[j].UsedPaths)
	})

	return entries
}

func (p *PathSelectionStore) Get(id string) PeerPathEntry {
	return p.data[id]
}

func (p *PathSelectionStore) updatePeerEntryInStore(entry PeerPathEntry) {
	p.data[entry.PeerAddrStr] = entry
}

func removePathFromEntry(entry PeerPathEntry, pathIndex int) PeerPathEntry {
	log.Warn(entry.UsedPaths)
	entry.UsedPaths = append(entry.UsedPaths[:pathIndex], entry.UsedPaths[pathIndex+1:]...)
	log.Warn(entry.UsedPaths)
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
		conflictButPeerHasNotEnoughPaths := false
		for i, targetEntry := range potentialConflictingPeers {
			conflictingPathIndex := getPeerConflictPaths(path, targetEntry)
			if conflictingPathIndex >= 0 {

				if len(targetEntry.UsedPaths) <= len(entry.UsedPaths) {
					conflictButPeerHasNotEnoughPaths = true
					break
				}

				fmt.Printf("Removing index %d from len %d\n", conflictingPathIndex, len(potentialConflictingPeers))
				// Remove path from targetEntry
				targetEntry = removePathFromEntry(targetEntry, conflictingPathIndex)

				// Replace targetEntry in map
				p.updatePeerEntryInStore(targetEntry)
				potentialConflictingPeers[i] = targetEntry

				// Add to our entry
				entry.UsedPaths = append(entry.UsedPaths, path)
				conflictFound = true
				break
			}
		}

		if conflictButPeerHasNotEnoughPaths {
			continue
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
	//newEntries := make([]PeerPathEntry, len(p.data))
	//return newEntries
	return entries
}
