package overpass

import (
	"log"
	"math"
	"strings"
)

// Only a unique same-name station within this radius may be used when OSM
// does not supply an explicit stop_area association.
const stationMatchRadiusMeters = 300.0

type routeStopKey struct {
	routeID int64
	nodeID  int64
}

type stationMatch struct {
	station importedStation
	found   bool
}

type stationResolver struct {
	stations     []importedStation
	byID         map[int64]importedStation
	nodes        map[int64]element
	linked       map[int64]map[int64]bool
	lineKeys     map[int64]string
	lineStations map[string]map[int64]bool
	// Matches are scoped to a line's route: an ambiguous interchange must not
	// inherit a station chosen for a different line.
	matches map[routeStopKey]stationMatch
}

func newStationResolver(stations []importedStation, elements []element) *stationResolver {
	r := &stationResolver{
		stations:     stations,
		byID:         make(map[int64]importedStation),
		nodes:        make(map[int64]element),
		linked:       make(map[int64]map[int64]bool),
		lineKeys:     make(map[int64]string),
		lineStations: make(map[string]map[int64]bool),
		matches:      make(map[routeStopKey]stationMatch),
	}
	for _, station := range stations {
		r.byID[station.osmID] = station
	}
	masters := make(map[int64]element)
	for _, item := range elements {
		if item.Type == "node" {
			r.nodes[item.ID] = item
		}
		if item.Type != "relation" || item.Tags["type"] != "route_master" || item.Tags["route_master"] != "subway" {
			continue
		}
		for _, member := range item.Members {
			if member.Type == "relation" {
				previous, exists := masters[member.Ref]
				if !exists || item.ID < previous.ID {
					masters[member.Ref] = item
				}
			}
		}
	}
	for _, area := range elements {
		if area.Type != "relation" || area.Tags["public_transport"] != "stop_area" {
			continue
		}
		var stationIDs []int64
		for _, member := range area.Members {
			if _, ok := r.byID[member.Ref]; member.Type == "node" && ok {
				stationIDs = append(stationIDs, member.Ref)
			}
		}
		for _, member := range area.Members {
			if member.Type != "node" || len(stationIDs) == 0 {
				continue
			}
			if r.linked[member.Ref] == nil {
				r.linked[member.Ref] = make(map[int64]bool)
			}
			for _, id := range stationIDs {
				r.linked[member.Ref][id] = true
			}
		}
	}
	// Explicit station memberships from either direction provide line context
	// for resolving same-name stations at an interchange.
	for _, route := range elements {
		if route.Type != "relation" || route.Tags["route"] != "subway" {
			continue
		}
		key := routeLineKey(route, masters[route.ID])
		r.lineKeys[route.ID] = key
		if r.lineStations[key] == nil {
			r.lineStations[key] = make(map[int64]bool)
		}
		for _, member := range route.Members {
			if !isRouteStop(member) {
				continue
			}
			if _, ok := r.byID[member.Ref]; ok {
				r.lineStations[key][member.Ref] = true
			} else if linked := r.linked[member.Ref]; len(linked) == 1 {
				for id := range linked {
					r.lineStations[key][id] = true
				}
			}
		}
	}
	return r
}

func isRouteStop(member relationMember) bool {
	return member.Type == "node" && (member.Role == "" || member.Role == "stop" ||
		member.Role == "stop_entry_only" || member.Role == "stop_exit_only")
}

func (r *stationResolver) resolve(routeID, nodeID int64) (importedStation, bool) {
	key := routeStopKey{routeID: routeID, nodeID: nodeID}
	if match, cached := r.matches[key]; cached {
		return match.station, match.found
	}
	station, found := r.match(routeID, nodeID)
	r.matches[key] = stationMatch{station: station, found: found}
	if !found {
		log.Printf("OSM route %d: stop node %d has no unambiguous station match", routeID, nodeID)
	}
	return station, found
}

func (r *stationResolver) match(routeID, nodeID int64) (importedStation, bool) {
	if station, ok := r.byID[nodeID]; ok {
		return station, true
	}
	linked := r.linked[nodeID]
	if len(linked) == 1 {
		for id := range linked {
			return r.byID[id], true
		}
	}
	node, exists := r.nodes[nodeID]
	if !exists {
		return importedStation{}, false
	}
	name := stationNameKey(relationName(node))
	var candidates []importedStation
	for _, station := range r.stations {
		if len(linked) > 0 {
			if !linked[station.osmID] {
				continue
			}
		} else {
			if name == "" || stationNameKey(station.name) != name {
				continue
			}
			point := importedStation{lat: node.Lat, lon: node.Lon}
			meters := math.Sqrt(squaredGeoDistance(point, station)) * math.Pi / 180 * 6371000
			if meters > stationMatchRadiusMeters {
				continue
			}
		}
		candidates = append(candidates, station)
	}
	if len(candidates) == 1 {
		return candidates[0], true
	}
	// Never pick the nearest of several plausible stations on different lines.
	// Only explicit membership of this line may disambiguate them.
	var anchored []importedStation
	for _, station := range candidates {
		if r.lineStations[r.lineKeys[routeID]][station.osmID] {
			anchored = append(anchored, station)
		}
	}
	if len(anchored) == 1 {
		return anchored[0], true
	}
	return importedStation{}, false
}

func stationNameKey(name string) string {
	return strings.ToLower(strings.Join(strings.Fields(name), " "))
}

func (r *stationResolver) routeStations(route element) []importedStation {
	seen := make(map[int64]bool)
	ordered := make([]importedStation, 0)
	for _, member := range route.Members {
		if !isRouteStop(member) {
			continue
		}
		station, ok := r.resolve(route.ID, member.Ref)
		if ok && !seen[station.osmID] {
			seen[station.osmID] = true
			ordered = append(ordered, station)
		}
	}
	return ordered
}
