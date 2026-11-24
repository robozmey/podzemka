import { useEffect, useMemo } from 'react'
import {
  CircleMarker,
  MapContainer,
  Polyline,
  TileLayer,
  Tooltip,
  useMap,
} from 'react-leaflet'
import type { LatLngBoundsExpression } from 'leaflet'
import 'leaflet/dist/leaflet.css'

import type { MetroSchema, Station } from '../types/metro'
import './MetroMap.css'

type GeographicMetroMapProps = {
  schema: MetroSchema
  visibleStations: Station[]
}

function FitMapToStations({ stations }: { stations: Station[] }) {
  const map = useMap()
  const bounds = useMemo<LatLngBoundsExpression | null>(() => {
    const points = stations
      .filter((station) => Number.isFinite(station.lat) && Number.isFinite(station.lon))
      .map((station) => [station.lat!, station.lon!] as [number, number])
    return points.length > 0 ? points : null
  }, [stations])

  useEffect(() => {
    if (bounds) map.fitBounds(bounds, { padding: [28, 28], maxZoom: 14 })
  }, [bounds, map])

  return null
}

export function GeographicMetroMap({ schema, visibleStations }: GeographicMetroMapProps) {
  const stationById = new Map(visibleStations.map((station) => [station.id, station]))
  const center = visibleStations.find(
    (station) => Number.isFinite(station.lat) && Number.isFinite(station.lon),
  )

  if (!center || center.lat == null || center.lon == null) return null

  return (
    <div className="geographic-map">
      <MapContainer
        center={[center.lat, center.lon]}
        zoom={11}
        scrollWheelZoom
        attributionControl={false}
        className="geographic-map__leaflet"
      >
        <TileLayer
          attribution="&copy; OpenStreetMap contributors"
          url="https://tile.openstreetmap.org/{z}/{x}/{y}.png"
        />
        <FitMapToStations stations={visibleStations} />

        {schema.lines.map((line) => {
          const points = line.stations
            .map((id) => stationById.get(id))
            .filter((station): station is Station => (
              station !== undefined
              && Number.isFinite(station.lat)
              && Number.isFinite(station.lon)
            ))
            .map((station) => [station.lat!, station.lon!] as [number, number])

          return (
            <Polyline
              key={line.id}
              positions={points}
              pathOptions={{ color: line.color, weight: 5, opacity: 0.9 }}
            />
          )
        })}

        {visibleStations
          .filter((station) => Number.isFinite(station.lat) && Number.isFinite(station.lon))
          .map((station) => (
            <CircleMarker
              key={station.id}
              center={[station.lat!, station.lon!]}
              radius={6}
              pathOptions={{ color: '#172033', fillColor: '#ffffff', fillOpacity: 1, weight: 2 }}
            >
              <Tooltip>
                <div className="geographic-map__station-popup">
                  <strong>Станция: {station.name}</strong>
                  <div>Год открытия: {station.year ?? 'не указан'}</div>
                  <div>Доп. информация: {station.description || 'будет добавлена позже'}</div>
                </div>
              </Tooltip>
            </CircleMarker>
          ))}
      </MapContainer>
      <a
        className="geographic-map__credit"
        href="https://www.openstreetmap.org/copyright"
        target="_blank"
        rel="noreferrer"
      >
        © OpenStreetMap contributors
      </a>
    </div>
  )
}
