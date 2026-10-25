import { useEffect, useState } from 'react'
import { getCities, getMetroSchema } from './api/metro'
import { MetroMap } from './components/MetroMap'
import type { City, MetroSchema } from './types/metro'
import type { MetroCitySlug } from './components/YearTimeline'
import './styles/App.css'

function App() {
  const [cities, setCities] = useState<City[]>([])
  const [citySlug, setCitySlug] = useState('saint-petersburg')
  const [schema, setSchema] = useState<MetroSchema | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    const controller = new AbortController()

    getCities(controller.signal)
      .then((loadedCities) => {
        setCities(loadedCities)
        setCitySlug(loadedCities[0]?.slug ?? '')
      })
      .catch((reason: unknown) => {
        if (reason instanceof Error && reason.name !== 'AbortError') {
          setError(reason.message)
        }
      })

    return () => controller.abort()
  }, [])

  useEffect(() => {
    if (!citySlug) return

    const controller = new AbortController()
    setSchema(null)
    setError(null)

    getMetroSchema(citySlug, controller.signal)
      .then(setSchema)
      .catch((reason: unknown) => {
        if (reason instanceof Error && reason.name !== 'AbortError') {
          setError(reason.message)
        }
      })

    return () => controller.abort()
  }, [citySlug])

  return (
    <main className="app">


      <section>
        <header className="map-header">
          <h1>Схема метро</h1>

          <div className="city-picker">
            <label htmlFor="city-select">Город</label>
            <div className="city-picker__control">
              <select
                id="city-select"
                value={citySlug}
                onChange={(event) => setCitySlug(event.target.value)}
                disabled={cities.length === 0}
              >
                {cities.map((city) => (
                  <option key={city.id} value={city.slug}>
                    {city.name}
                  </option>
                ))}
              </select>
            </div>
          </div>
        </header>

        {error && <p className="message message--error">{error}</p>}
        {!schema && !error && <p className="message">Загрузка схемы...</p>}
        {schema && (
          <MetroMap
            schema={schema}
            city={citySlug as MetroCitySlug}
          />
        )}
      </section>
    </main>
  )
}

export default App
