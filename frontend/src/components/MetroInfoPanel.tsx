import type { MetroSchema, Station, MetroLine } from '../types/metro'
import './MetroInfoPanel.css'

type Props = {
  schema: MetroSchema
  selectedLines: MetroLine[]
  selectedStation: Station | null
  onClearSelection: () => void
}

export function MetroInfoPanel({
  schema,
  selectedLines,
  selectedStation,
  onClearSelection,
}: Props) {
  return (
    <div className="metro-info-panel">
      <div className="metro-info-card">
        <span className="metro-info-card__label">Город</span>
        <h3>{schema.city.name}</h3>
        <p>Станций: {schema.stations.length}</p>
        <p>Линий: {schema.lines.length}</p>
      </div>

      {selectedLines.length > 0 && (
        <div className="metro-info-card">
          {!selectedStation && (
            <button
              type="button"
              className="metro-info-card__close"
              onClick={onClearSelection}
              aria-label="Снять выделение линии"
              title="Закрыть"
            >
              ×
            </button>
          )}
          <span className="metro-info-card__label">Линия</span>

          {selectedLines.map((line) => (
            <div key={line.id}>
              <h3>
                <span
                  className="metro-info-card__line-color"
                  style={{ backgroundColor: line.color }}
                />
                {line.name}
              </h3>
              <p>Станций: {line.stations.length}</p>
            </div>
          ))}
        </div>
      )}

      {selectedStation && (
        <div className="metro-info-card">
          <button
            type="button"
            className="metro-info-card__close"
            onClick={onClearSelection}
            aria-label="Снять выделение"
          >
            ×
          </button>

          <span className="metro-info-card__label">Станция</span>
          <h3>{selectedStation.name}</h3>
          <p>
            Год открытия: {selectedStation.year || 'не указан'}
          </p>
        </div>
      )}
    </div>
  )
}