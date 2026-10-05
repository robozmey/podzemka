import type { MetroSchema, Station } from '../types/metro'
import './MetroMap.css'

type MetroMapProps = {
  schema: MetroSchema
}

export function MetroMap({ schema }: MetroMapProps) {
  const stationById = new Map(schema.stations.map((station) => [station.id, station]))

  return (
    <svg
      className="metro-map"
      viewBox="0 0 700 420"
      role="img"
      aria-label="Схема Санкт-Петербургского метро"
    >
      {schema.lines.map((line) => {
        const stations = line.stations
          .map((stationId) => stationById.get(stationId))
          .filter((station): station is Station => station !== undefined)

        return (
          <polyline
            key={line.id}
            points={stations.map(({ x, y }) => `${x},${y}`).join(' ')}
            fill="none"
            stroke={line.color}
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth="12"
          />
        )
      })}

      {schema.stations.map((station) => (
        <g key={station.id} className="station">
          <circle cx={station.x} cy={station.y} r="10" />
          <text x={station.x} y={station.y + 28}>{station.name}</text>
        </g>
      ))}
    </svg>
  )
}
