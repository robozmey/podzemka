package overpass

import (
	"fmt"
	"sort"
	"strings"
)

type importedLine struct {
	name       string
	color      string
	stations   []importedStation
	routes     []element
	edges      [][2]int64
	stationSet map[int64]bool
	edgeSet    map[[2]int64]bool
}

// collectLines uses route membership as the source of line composition. Exports
// without subway routes retain support for the older, station-only format.
func collectLines(stations []importedStation, elements []element, resolver *stationResolver) []importedLine {
	var routes, masters []element
	for _, item := range elements {
		if item.Type != "relation" {
			continue
		}
		if item.Tags["route"] == "subway" {
			routes = append(routes, item)
		}
		if item.Tags["type"] == "route_master" && item.Tags["route_master"] == "subway" {
			masters = append(masters, item)
		}
	}
	if len(routes) == 0 {
		return stationOnlyLines(stations)
	}
	sort.Slice(routes, func(i, j int) bool { return routes[i].ID < routes[j].ID })
	sort.Slice(masters, func(i, j int) bool { return masters[i].ID < masters[j].ID })
	masterByRoute := make(map[int64]element)
	for _, master := range masters {
		for _, member := range master.Members {
			if member.Type == "relation" {
				if _, exists := masterByRoute[member.Ref]; !exists {
					masterByRoute[member.Ref] = master
				}
			}
		}
	}
	groups := make(map[string]*importedLine)
	for _, route := range routes {
		master, hasMaster := masterByRoute[route.ID]
		key := routeLineKey(route, master)

		line := groups[key]
		if line == nil {
			// An empty slice means routes were processed but no edges were
			// resolved. Nil is reserved for older station-only exports.
			line = &importedLine{edges: make([][2]int64, 0)}
			groups[key] = line
		}
		if line.name == "" {
			line.name = relationName(route)
		}
		if line.color == "" {
			line.color = normalizeColor(route.Tags["colour"])
		}
		if hasMaster {
			if name := relationName(master); name != "" {
				line.name = name
			}
			if color := normalizeColor(master.Tags["colour"]); color != "" {
				line.color = color
			}
		}
		line.routes = append(line.routes, route)
		line.addRoute(route, resolver)
	}
	keys := make([]string, 0, len(groups))
	for key := range groups {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	lines := make([]importedLine, 0, len(keys))
	for _, key := range keys {
		line := groups[key]
		if len(line.stations) == 0 {
			continue
		}
		if line.name == "" {
			line.name = "Линия " + strings.TrimSpace(line.routes[0].Tags["ref"])
			if line.name == "Линия " {
				line.name = fmt.Sprintf("Линия OSM %d", line.routes[0].ID)
			}
		}
		if line.color == "" {
			line.color = "#808080"
		}
		lines = append(lines, *line)
	}
	return lines
}

func (line *importedLine) addStation(station importedStation) {
	if line.stationSet == nil {
		line.stationSet = make(map[int64]bool)
	}
	if !line.stationSet[station.osmID] {
		line.stationSet[station.osmID] = true
		line.stations = append(line.stations, station)
	}
}

func (line *importedLine) addEdge(from, to importedStation) {
	line.addStation(from)
	line.addStation(to)
	if from.osmID == to.osmID {
		return
	}
	edge := [2]int64{from.osmID, to.osmID}
	if edge[0] > edge[1] {
		edge[0], edge[1] = edge[1], edge[0]
	}
	if line.edgeSet == nil {
		line.edgeSet = make(map[[2]int64]bool)
	}
	if !line.edgeSet[edge] {
		line.edgeSet[edge] = true
		line.edges = append(line.edges, edge)
	}
}

func (line *importedLine) addRoute(route element, resolver *stationResolver) {
	var previous importedStation
	havePrevious := false
	for _, member := range route.Members {
		if !isRouteStop(member) {
			continue
		}
		station, ok := resolver.resolve(route.ID, member.Ref)
		if !ok {
			// resolve logs the missing/ambiguous stop. Do not invent a link
			// across it, or connect endpoints of separate route variants.
			havePrevious = false
			continue
		}
		line.addStation(station)
		if havePrevious {
			line.addEdge(previous, station)
		}
		// Keep repeated visits until edges have been built: the last visit
		// to the first station is what closes a circular route.
		previous, havePrevious = station, true
	}
}

func relationName(item element) string {
	if name := strings.TrimSpace(item.Tags["name:ru"]); name != "" {
		return name
	}
	return strings.TrimSpace(item.Tags["name"])
}

func stationOnlyLines(stations []importedStation) []importedLine {
	groups := make(map[string][]importedStation)
	for _, station := range stations {
		if station.color != "" {
			groups[station.color] = append(groups[station.color], station)
		}
	}
	colors := make([]string, 0, len(groups))
	for color := range groups {
		colors = append(colors, color)
	}
	sort.Strings(colors)
	lines := make([]importedLine, 0, len(colors))
	for _, color := range colors {
		name := "Линия " + color
		for _, known := range lineColors {
			if known.color == color {
				name = known.name
				break
			}
		}
		lines = append(lines, importedLine{name: name, color: color, stations: groups[color]})
	}
	return lines
}

// A missing ref must not merge unrelated routes merely because colors match.
func routeLineKey(route, master element) string {
	if master.ID != 0 {
		return fmt.Sprintf("master:%d", master.ID)
	}
	if ref := strings.TrimSpace(route.Tags["ref"]); ref != "" {
		return "ref:" + strings.TrimSpace(route.Tags["network"]) + ":" + ref
	}
	return fmt.Sprintf("route:%d", route.ID)
}
