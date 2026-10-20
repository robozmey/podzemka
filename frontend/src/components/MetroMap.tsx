import { useEffect, useState } from 'react'

import YearTimeline, {
  CITY_YEAR_RANGES,
  type MetroCitySlug,
} from './YearTimeline'
import type { MetroSchema, Station } from '../types/metro'
import './MetroMap.css'

type MetroMapProps = {
  schema: MetroSchema
  city: MetroCitySlug
}

export function MetroMap({ schema, city }: MetroMapProps) {
  const [hoveredStation, setHoveredStation] = useState<Station | null>(null)
  const [selectedYear, setSelectedYear] = useState(
    CITY_YEAR_RANGES[city].start,
  )

  useEffect(() => {
    setSelectedYear(CITY_YEAR_RANGES[city].start)
    setHoveredStation(null)
  }, [city])

  const range = CITY_YEAR_RANGES[city]

  const year = Math.max(
    range.start,
    Math.min(range.end, selectedYear),
  )

  const visibleStations = schema.stations.filter(
    (station) => station.year == null || station.year <= year,
  )
  const stationById = new Map(visibleStations.map((station) => [station.id, station]),)

  return (
    <div className="metro-map-layout">
      <YearTimeline
        city={city}
        year={year}
        onYearChange={(nextYear: number) => {
          setSelectedYear(nextYear)
          setHoveredStation(null)
        }}
      />

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
          const station = visibleStations.find((item) => item.id === stationId,)
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

        {visibleStations.map((station) => (
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
              <p>Год открытия: {hoveredStation.year || "не указан"}.</p>
              <p>Здесь можно разместить описание станции.</p>
            </div>
          </foreignObject>
        )}
      </svg>
    </div>
  )
}
