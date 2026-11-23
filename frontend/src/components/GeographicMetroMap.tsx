
import type { MetroSchema, Station } from '../types/metro'

type Props = {
  schema: MetroSchema
  visibleStations: Station[]
}

export function GeographicMetroMap({ schema, visibleStations }: Props) {
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

  return (
    <svg
      className="metro-map"
      viewBox={`0 0 ${width} ${height}`}
      role="img"
      aria-label={`Географическая схема метро: ${schema.city.name}`}
    >
      {schema.lines.map(line => {
        const points = line.stations
          .map(id => byId.get(id))
          .filter((s): s is Station => s != null)
          .map(s => `${x(s)},${y(s)}`)
          .join(' ')

        return (
          <polyline
            key={line.id}
            points={points}
            fill="none"
            stroke={line.color}
            strokeWidth={3}
          />
        )
      })}

      {stations.map(station => (
        <g key={station.id}>
          <circle
            cx={x(station)}
            cy={y(station)}
            r={3}
            fill="white"
            stroke="black"
          />
          <text
            x={x(station)}
            y={y(station) - 7}
            fontSize={9}
            fill="currentColor"
            textAnchor="middle"
          >
            {station.name}
          </text>
        </g>
      ))}
    </svg>
  )
}
