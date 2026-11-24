import type { MetroLine, Station } from '../types/metro'

export function getLineEdges(line: MetroLine): [number, number][] {
  if (line.edges != null) return line.edges

  // Derive legacy connections before filtering stations: hiding a station
  // must not create a new connection between its former neighbors.
  return line.stations.slice(1).map((id, index) => [line.stations[index], id])
}

export function getVisibleLineEdges(
  line: MetroLine,
  stationsById: ReadonlyMap<number, Station>,
): [Station, Station][] {
  return getLineEdges(line).flatMap(([fromId, toId]) => {
    const from = stationsById.get(fromId)
    const to = stationsById.get(toId)
    return from && to ? [[from, to] as [Station, Station]] : []
  })
}
