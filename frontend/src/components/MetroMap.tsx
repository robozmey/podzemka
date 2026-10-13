import { useState } from 'react'
import type { MetroSchema, Station } from '../types/metro'
import './MetroMap.css'

type MetroMapProps = {
  schema: MetroSchema
}

export function MetroMap({ schema }: MetroMapProps) {
  const [hoveredStation, setHoveredStation] = useState<Station | null>(null)
  const stationById = new Map(schema.stations.map((station) => [station.id, station]))

  return (
    <svg
      className="metro-map"
      viewBox="0 0 700 420"
      role="img"
      aria-label="Схема Санкт-Петербургского метро"
      onPointerOver={(event) => {
        if (!(event.target instanceof Element)) return

        const group = event.target.closest<SVGGElement>('[data-station-id]')
        if (!group) return

        const stationId = Number(group.dataset.stationId)
        const station = schema.stations.find((item) => item.id === stationId)
        if (station) setHoveredStation(station)
      }}
      onPointerLeave={() => setHoveredStation(null)}
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
        <g key={station.id} className="station" data-station-id={station.id}>
          <circle cx={station.x} cy={station.y} r="10" />
          <text x={station.x} y={station.y + 28}>{station.name}</text>
        </g>
      ))}

      {hoveredStation && (
        <foreignObject
          className="station__tooltip"
          x={hoveredStation.x > 460 ? hoveredStation.x - 238 : hoveredStation.x + 18}
          y={Math.max(10, Math.min(hoveredStation.y - 45, 280))}
          width="220"
          height="130"
          onPointerLeave={() => setHoveredStation(null)}
        >
          <div xmlns="http://www.w3.org/1999/xhtml" className="station__tooltip-content">
            <strong>{hoveredStation.name}</strong>
            <p>
              Линии: {schema.lines
                .filter((line) => line.stations.includes(hoveredStation.id))
                .map((line) => line.name)
                .join(', ') || 'не указаны'}
            </p>
            <p>Здесь можно разместить описание станции.</p>
          </div>
        </foreignObject>
      )}
    </svg>
  )
}
