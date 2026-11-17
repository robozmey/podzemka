import { useEffect, useState } from 'react'
import { getCities, getMetroSchema } from './api/metro'
import { MetroMap } from './components/MetroMap'
import { AdminCityPanel } from './features/AdminCityPanel'
import type { City, MetroSchema } from './types/metro'
import './styles/App.css'

function App() {
  const [cities, setCities] = useState<City[]>([])
  const [citySlug, setCitySlug] = useState('saint-petersburg')
  const [schema, setSchema] = useState<MetroSchema | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [adminOpen, setAdminOpen] = useState(false)

  const loadCities = (signal?: AbortSignal) => getCities(signal).then((loaded) => {
    setCities(loaded)
    return loaded
  })

  useEffect(() => {
    const controller = new AbortController()
    loadCities(controller.signal)
      .then((loaded) => setCitySlug((current) => loaded.some((city) => city.slug === current) ? current : (loaded[0]?.slug ?? '')))
      .catch((reason: unknown) => {
        if (reason instanceof Error && reason.name !== 'AbortError') setError(reason.message)
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
        if (reason instanceof Error && reason.name !== 'AbortError') setError(reason.message)
      })
    return () => controller.abort()
  }, [citySlug])

  const handleCityCreated = async (slug: string) => {
    const loaded = await loadCities()
    setCities(loaded)
    setCitySlug(slug)
  }

  return (
    <main className="app">
      <section>
        <header className="map-header">
          <h1>Схема метро</h1>
          <div className="city-picker">
            <label htmlFor="city-select">Город</label>
            <div className="city-picker__control">
              <select id="city-select" value={citySlug} onChange={(event) => setCitySlug(event.target.value)} disabled={cities.length === 0}>
                {cities.map((city) => <option key={city.id} value={city.slug}>{city.name}</option>)}
              </select>
            </div>
          </div>
          <button className="admin-trigger" type="button" onClick={() => setAdminOpen(true)}>
            Администрирование
          </button>
        </header>

        <AdminCityPanel
          open={adminOpen}
          onClose={() => setAdminOpen(false)}
          onCityCreated={handleCityCreated}
        />

        {error && <p className="message message--error">{error}</p>}
        {!schema && !error && <p className="message">Загрузка схемы...</p>}
        {schema && <MetroMap schema={schema} />}
      </section>
    </main>
  )
}

export default App
