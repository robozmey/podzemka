import { useEffect, useLayoutEffect, useRef, useState } from 'react'

import YearTimeline, {
  getCityTimeline,
} from './YearTimeline'
import type { MetroSchema, Station } from '../types/metro'
import './MetroMap.css'
import { arrangeLabels, type LabelPlacement } from './labelLayout'
import { GeographicMetroMap } from './GeographicMetroMap'

type MetroMapProps = {
  schema: MetroSchema
}

export function MetroMap({ schema }: MetroMapProps) {
  const timeline = getCityTimeline(schema)
  const { range } = timeline
  const [hoveredStation, setHoveredStation] = useState<Station | null>(null)
  const [selectedYear, setSelectedYear] = useState(
    range.end,
  )
  const [viewMode, setViewMode] = useState<'schematic' | 'geographic'>('geographic')
  const svgRef = useRef<SVGSVGElement>(null)
  const [labelPlacements, setLabelPlacements] = useState<LabelPlacement[]>([])

  useEffect(() => {
    setSelectedYear(range.end)
    setHoveredStation(null)
  }, [schema])


  const year = Math.max(
    range.start,
    Math.min(range.end, selectedYear),
  )

  const visibleStations = schema.stations.filter(
    (station) => station.year == null || station.year <= year,
  )
  const hasGeographicCoordinates = schema.stations.length > 0
    && schema.stations.every((station) => (
      Number.isFinite(station.lat)
      && Number.isFinite(station.lon)
    ))
  const stationById = new Map(visibleStations.map((station) => [station.id, station]),)
  const coordinateBounds = schema.stations.reduce(
    (bounds, station) => ({
      minX: Math.min(bounds.minX, station.x),
      maxX: Math.max(bounds.maxX, station.x),
      minY: Math.min(bounds.minY, station.y),
      maxY: Math.max(bounds.maxY, station.y),
    }),
    {
      minX: Number.POSITIVE_INFINITY,
      maxX: Number.NEGATIVE_INFINITY,
      minY: Number.POSITIVE_INFINITY,
      maxY: Number.NEGATIVE_INFINITY,
    },
  )
  const coordinateWidth = Math.max(1, coordinateBounds.maxX - coordinateBounds.minX)
  const coordinateHeight = Math.max(1, coordinateBounds.maxY - coordinateBounds.minY)
  const coordinateScale = schema.city.slug === 'saint-petersburg'
    ? 1
    : Math.min(640 / coordinateWidth, 360 / coordinateHeight)
  const coordinateTransform = schema.city.slug === 'saint-petersburg'
    ? undefined
    : `translate(${(700 - coordinateWidth * coordinateScale) / 2 - coordinateBounds.minX * coordinateScale} ${(420 - coordinateHeight * coordinateScale) / 2 - coordinateBounds.minY * coordinateScale}) scale(${coordinateScale})`
  useLayoutEffect(() => {
    const svg = svgRef.current
    if (!svg) return

    const textElements = svg.querySelectorAll<SVGTextElement>(
      '.station__label',
    )

    const textById = new Map(
      Array.from(textElements).map((element) => [
        Number(element.dataset.stationId),
        element,
      ]),
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
      lineSegments,
    )

    setLabelPlacements(result.placements)
  }, [schema, year])

  const placementById = new Map(
    labelPlacements.map((placement) => [
      placement.id,
      placement,
    ]),
  )
  return (
    <div className="metro-map-layout">
      {timeline.hasYears && (
        <YearTimeline
          timeline={timeline}
          year={year}
          onYearChange={(nextYear: number) => {
            setSelectedYear(nextYear)
            setHoveredStation(null)
          }}
        />
      )}

      {hasGeographicCoordinates && (
        <div className="metro-map__view-switcher" role="group" aria-label="Режим отображения карты">
          <button
            type="button"
            className={viewMode === 'geographic' ? 'is-active' : ''}
            onClick={() => setViewMode('geographic')}
          >
            Географическая карта
          </button>
          <button
            type="button"
            className={viewMode === 'schematic' ? 'is-active' : ''}
            onClick={() => setViewMode('schematic')}
          >
            Схема метро
          </button>
        </div>
      )}

      {hasGeographicCoordinates && viewMode === 'geographic' && (
        <GeographicMetroMap
          schema={schema}
          visibleStations={visibleStations}
        />
      )}

      <svg
        ref={svgRef}
        style={{ display: viewMode === 'geographic' && hasGeographicCoordinates ? 'none' : undefined }}
        className="metro-map"
        viewBox="0 0 700 420"
        role="img"
        aria-label={`Схема метро: ${schema.city.name}`}
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
        <g transform={coordinateTransform}>
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
              strokeWidth="2.4"
            />
          )
        })}

        {visibleStations.map((station) => {
          const placement = placementById.get(station.id)

          return (
            <g
              key={station.id}
              className="station"
              data-station-id={station.id}
            >
              <circle cx={station.x} cy={station.y} r="2" />

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

        {hoveredStation && (
          <foreignObject
            className="station__tooltip"
              x={hoveredStation.x > 460 ? hoveredStation.x - 198 : hoveredStation.x + 14}
            y={Math.max(10, Math.min(hoveredStation.y - 45, 280))}
              width="180"
              height="105"
            onPointerLeave={() => setHoveredStation(null)}
          >
            <div className="station__tooltip-content">
              <strong>Станция: {hoveredStation.name}</strong>
              <p>Год открытия: {hoveredStation.year ?? 'не указан'}</p>
              <p>
                Линии: {schema.lines
                  .filter((line) => line.stations.includes(hoveredStation.id))
                  .map((line) => line.name)
                  .join(', ') || 'не указаны'}
              </p>
              <p>Доп. информация: {hoveredStation.description || 'будет добавлена позже'}.</p>
            </div>
          </foreignObject>
        )}
        </g>
      </svg>
    </div>
  )
}
