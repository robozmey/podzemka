
import { useEffect, useLayoutEffect, useRef, useState } from 'react'
import YearTimeline, { getCityTimeline } from './YearTimeline'
import type { MetroSchema, Station } from '../types/metro'
import './MetroMap.css'
import { arrangeLabels, type LabelPlacement } from './labelLayout'
import { MetroInfoPanel } from './MetroInfoPanel'
import { GeographicMetroMap } from './GeographicMetroMap'

type MetroMapProps = {
  schema: MetroSchema
}

export function MetroMap({ schema }: MetroMapProps) {
  const timeline = getCityTimeline(schema)
  const { range } = timeline

  const [hoveredStation, setHoveredStation] = useState<Station | null>(null)
  const [selectedYear, setSelectedYear] = useState(range.end)
  const [selectedLineId, setSelectedLineId] = useState<number | null>(null)
  const [selectedStationId, setSelectedStationId] = useState<number | null>(null)
  const [labelPlacements, setLabelPlacements] = useState<LabelPlacement[]>([])

  const svgRef = useRef<SVGSVGElement>(null)

  const selectedLine = schema.lines.find(
    (line) => line.id === selectedLineId
  )

  const selectedStation = schema.stations.find(
    (station) => station.id === selectedStationId
  ) ?? null

  const selectedLines = selectedStation
    ? schema.lines.filter((line) =>
        line.stations.includes(selectedStation.id)
      )
    : selectedLine
      ? [selectedLine]
      : []

  const highlightedLineIds = new Set(
    selectedLines.map((line) => line.id)
  )

  useEffect(() => {
    setSelectedYear(range.end)
    setHoveredStation(null)
    setSelectedLineId(null)
    setSelectedStationId(null)
  }, [schema])

  const year = Math.max(
    range.start,
    Math.min(range.end, selectedYear)
  )

  const visibleStations = schema.stations.filter(
    (station) => station.year == null || station.year <= year
  )

  const hasGeographicCoordinates =
    schema.stations.length > 0 &&
    schema.stations.every((station) => (
      Number.isFinite(station.lat) &&
      Number.isFinite(station.lon) &&
      station.lat! >= -90 &&
      station.lat! <= 90 &&
      station.lon! >= -180 &&
      station.lon! <= 180
    ))

  const stationById = new Map(
    visibleStations.map((station) => [station.id, station])
  )

  useLayoutEffect(() => {
    const svg = svgRef.current
    if (!svg || hasGeographicCoordinates) return

    const textElements = svg.querySelectorAll<SVGTextElement>(
      '.station__label'
    )

    const textById = new Map(
      Array.from(textElements).map((element) => [
        Number(element.dataset.stationId),
        element,
      ])
    )

    const labels = visibleStations.flatMap((station) => {
      const element = textById.get(station.id)
      if (!element) return []

      const bounds = element.getBBox()
      const originalX = station.x
      const originalY = station.y + 5.6

      const baselineOffsetY =
        bounds.y - Number(element.getAttribute('y'))

      return [{
        id: station.id,
        stationX: station.x,
        stationY: station.y,
        width: bounds.width,
        height: bounds.height,
        originalX,
        originalY,
        originalRect: {
          x: originalX - bounds.width / 2,
          y: originalY + baselineOffsetY,
          width: bounds.width,
          height: bounds.height,
        },
        baselineOffsetY,
      }]
    })

    const stationObstacles = visibleStations.map((station) => ({
      id: station.id,
      x: station.x,
      y: station.y,
      radius: 2,
    }))

    const lineSegments = schema.lines.flatMap((line) => {
      const points = line.stations
        .map((id) => stationById.get(id))
        .filter((station): station is Station => station !== undefined)

      return points.slice(1).map((point, index) => ({
        x1: points[index].x,
        y1: points[index].y,
        x2: point.x,
        y2: point.y,
      }))
    })

    const result = arrangeLabels(
      labels,
      stationObstacles,
      lineSegments
    )

    setLabelPlacements(result.placements)
  }, [schema, year, hasGeographicCoordinates])

  const placementById = new Map(
    labelPlacements.map((placement) => [
      placement.id,
      placement,
    ])
  )

  const clearSelection = () => {
    setSelectedLineId(null)
    setSelectedStationId(null)
  }

  const toggleLine = (id: number) => {
    setSelectedStationId(null)
    setSelectedLineId((current) => current === id ? null : id)
  }

  const toggleStation = (id: number) => {
    setSelectedLineId(null)
    setHoveredStation(null)
    setSelectedStationId((current) => current === id ? null : id)
  }

  return (
    <div className="metro-view">
      <MetroInfoPanel
        schema={schema}
        selectedLines={selectedLines}
        selectedStation={selectedStation}
        onClearSelection={clearSelection}
      />

      <div className="metro-map-layout">
        {timeline.hasYears && (
          <YearTimeline
            timeline={timeline}
            year={year}
            onYearChange={(nextYear: number) => {
              setSelectedYear(nextYear)
              setSelectedStationId((current) => {
                const station = schema.stations.find(
                  (item) => item.id === current
                )

                return station?.year != null &&
                  station.year > nextYear
                  ? null
                  : current
              })
              setHoveredStation(null)
            }}
          />
        )}

        <div className="metro-map-content">
          {hasGeographicCoordinates && (
            <GeographicMetroMap
              schema={schema}
              visibleStations={visibleStations}
            />
          )}

          <svg
            ref={svgRef}
            style={{
              display: hasGeographicCoordinates ? 'none' : undefined,
            }}
            className="metro-map"
            viewBox="0 0 700 420"
            role="img"
            aria-label={`Схема метро: ${schema.city.name}`}
            onPointerOver={(event) => {
              if (!(event.target instanceof Element)) return

              const group = event.target.closest<SVGGElement>(
                '[data-station-id]'
              )
              if (!group) return

              const stationId = Number(group.dataset.stationId)
              const station = visibleStations.find(
                (item) => item.id === stationId
              )

              if (station) setHoveredStation(station)
            }}
            onPointerLeave={() => setHoveredStation(null)}
          >
            {schema.city.slug === 'saint-petersburg' && (
              <image
                className="metro-map__background"
                href="/maps/saint-petersburg-rivers.png"
                x="0"
                y="0"
                width="700"
                height="420"
                preserveAspectRatio="xMidYMid slice"
              />
            )}

            {schema.lines.map((line) => {
              const stations = line.stations
                .map((stationId) => stationById.get(stationId))
                .filter(
                  (station): station is Station =>
                    station !== undefined
                )

              const points = stations
                .map(({ x, y }) => `${x},${y}`)
                .join(' ')

              return (
                <g key={line.id}>
                  <polyline
                    points={points}
                    fill="none"
                    stroke={line.color}
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={
                      highlightedLineIds.has(line.id) ? 4 : 2.4
                    }
                    opacity={
                      highlightedLineIds.size > 0 &&
                      !highlightedLineIds.has(line.id)
                        ? 0.4
                        : 1
                    }
                    pointerEvents="none"
                  />

                  <polyline
                    points={points}
                    fill="none"
                    stroke="transparent"
                    strokeWidth="12"
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    pointerEvents="stroke"
                    className="metro-line-hitbox"
                    role="button"
                    tabIndex={0}
                    aria-label={`Информация о линии ${line.name}`}
                    onClick={() => toggleLine(line.id)}
                    onKeyDown={(event) => {
                      if (
                        event.key === 'Enter' ||
                        event.key === ' '
                      ) {
                        event.preventDefault()
                        toggleLine(line.id)
                      }
                    }}
                  />
                </g>
              )
            })}

            {visibleStations.map((station) => {
              const placement = placementById.get(station.id)

              return (
                <g
                  key={station.id}
                  className={`station ${
                    selectedStationId === station.id
                      ? 'station--selected'
                      : ''
                  }`}
                  data-station-id={station.id}
                  role="button"
                  tabIndex={0}
                  aria-label={`Выбрать станцию ${station.name}`}
                  onClick={() => toggleStation(station.id)}
                  onKeyDown={(event) => {
                    if (
                      event.key === 'Enter' ||
                      event.key === ' '
                    ) {
                      event.preventDefault()
                      toggleStation(station.id)
                    }
                  }}
                >
                  <circle
                    cx={station.x}
                    cy={station.y}
                    r="2"
                  />

                  <text
                    className="station__label"
                    data-station-id={station.id}
                    x={placement?.x ?? station.x}
                    y={placement?.y ?? station.y + 5.6}
                    textAnchor={placement?.anchor ?? 'middle'}
                  >
                    {station.name}
                  </text>
                </g>
              )
            })}

            {hoveredStation &&
              hoveredStation.id !== selectedStationId && (
                <foreignObject
                  className="station__tooltip"
                  x={
                    hoveredStation.x > 460
                      ? hoveredStation.x - 198
                      : hoveredStation.x + 18
                  }
                  y={Math.max(
                    10,
                    Math.min(hoveredStation.y - 45, 280)
                  )}
                  width="180"
                  height="85"
                  onPointerLeave={() => setHoveredStation(null)}
                >
                  <div className="station__tooltip-content">
                    <strong>{hoveredStation.name}</strong>

                    <p>
                      {schema.lines
                        .filter((line) =>
                          line.stations.includes(hoveredStation.id)
                        )
                        .map((line) => line.name)
                        .join(', ') || 'Линия не указана'}
                    </p>

                    {hoveredStation.year != null &&
                      hoveredStation.year > 0 && (
                        <p>
                          Открыта в {hoveredStation.year} году
                        </p>
                      )}
                  </div>
                </foreignObject>
              )}
          </svg>
        </div>
      </div>
    </div>
  )
}
