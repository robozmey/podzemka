
import type { MetroSchema, Station } from '../types/metro'
import { getVisibleLineEdges } from './metroEdges'

type Props = {
  schema: MetroSchema
  visibleStations: Station[]
  selectedStationId: number | null
  selectedLineId: number | null
  highlightedLineIds: ReadonlySet<number>
  hoveredStation: Station | null
  onStationClick: (id: number) => void
  onLineClick: (id: number) => void
  onStationHover: (station: Station | null) => void
}

export function GeographicMetroMap({
  schema, visibleStations, selectedStationId, selectedLineId,
  highlightedLineIds, hoveredStation, onStationClick, onLineClick, onStationHover,
}: Props) {
  const stations = visibleStations.filter(
    s => s.lat != null && s.lon != null
  )

  if (stations.length === 0) return null

  const lats = stations.map(s => s.lat!)
  const lons = stations.map(s => s.lon!)

  const minLat = Math.min(...lats)
  const maxLat = Math.max(...lats)
  const minLon = Math.min(...lons)
  const maxLon = Math.max(...lons)

  const width = 700
  const height = 420
  const padding = 30

  const scaleX = (width - 2 * padding) /
    Math.max(maxLon - minLon, 0.000001)
  const scaleY = (height - 2 * padding) /
    Math.max(maxLat - minLat, 0.000001)
  const scale = Math.min(scaleX, scaleY)

  const x = (s: Station) =>
    width / 2 + (s.lon! - (minLon + maxLon) / 2) * scale

  const y = (s: Station) =>
    height / 2 - (s.lat! - (minLat + maxLat) / 2) * scale

  const byId = new Map(stations.map(s => [s.id, s]))
  const hovered = hoveredStation ? byId.get(hoveredStation.id) : undefined

  return (
    <svg
      className="metro-map geographic-metro-map"
      viewBox={`0 0 ${width} ${height}`}
      role="group"
      aria-label={`Географическая схема метро: ${schema.city.name}`}
      onPointerLeave={() => onStationHover(null)}
    >
      {schema.lines.map(line => {
        const edges = getVisibleLineEdges(line, byId)
        if (edges.length === 0) return null

        const path = edges
          .map(([from, to]) => `M ${x(from)},${y(from)} L ${x(to)},${y(to)}`)
          .join(' ')

        return (
          <g key={line.id}>
            <path
              d={path}
              fill="none"
              stroke={line.color}
              strokeWidth={highlightedLineIds.has(line.id) ? 4 : 3}
              strokeLinecap="round"
              opacity={highlightedLineIds.size > 0 && !highlightedLineIds.has(line.id) ? 0.2 : 1}
              pointerEvents="none"
            />
            <path
              d={path}
              fill="none"
              stroke="transparent"
              strokeWidth={12}
              strokeLinecap="round"
              pointerEvents="stroke"
              className="metro-line-hitbox"
              role="button"
              tabIndex={0}
              aria-label={`Информация о линии ${line.name}`}
              aria-pressed={selectedLineId === line.id}
              onClick={() => onLineClick(line.id)}
              onKeyDown={event => {
                if (event.key === 'Enter' || event.key === ' ') {
                  event.preventDefault()
                  onLineClick(line.id)
                }
              }}
            />
          </g>
        )
      })}

      {stations.map(station => (
        <g
          key={station.id}
          className={`station ${selectedStationId === station.id ? 'station--selected' : ''}`}
          role="button"
          tabIndex={0}
          aria-label={`Выбрать станцию ${station.name}`}
          aria-pressed={selectedStationId === station.id}
          onClick={() => onStationClick(station.id)}
          onPointerEnter={() => onStationHover(station)}
          onPointerLeave={() => onStationHover(null)}
          onFocus={() => onStationHover(station)}
          onBlur={() => onStationHover(null)}
          onKeyDown={event => {
            if (event.key === 'Enter' || event.key === ' ') {
              event.preventDefault()
              onStationClick(station.id)
            }
          }}
        >
          <circle
            cx={x(station)}
            cy={y(station)}
            r={3}
            fill="white"
            stroke="black"
          />
          <text
            className={`station__label ${
              hovered?.id !== station.id &&
              selectedStationId === station.id
                ? 'station__label--visible' : ''
            }`}
            x={x(station)}
            y={y(station) - 7}
            fill="currentColor"
            textAnchor="middle"
          >
            {station.name}
          </text>
        </g>
      ))}

      {hovered && (
        <text
          className="station__hover-label"
          x={x(hovered) + (x(hovered) < width / 3 ? 7 : x(hovered) > width * 2 / 3 ? -7 : 0)}
          y={y(hovered) - 7}
          textAnchor={x(hovered) < width / 3 ? 'start' : x(hovered) > width * 2 / 3 ? 'end' : 'middle'}
        >
          {hovered.name}
        </text>
      )}
    </svg>
  )
}
