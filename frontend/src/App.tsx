import { useEffect, useState } from 'react'
import { getCities, getMetroSchema } from './api/metro'
import { MetroMap } from './components/MetroMap'
import { AdminCityPanel } from './features/AdminCityPanel'
import { CitySelector } from './components/CitySelector'
import type { City, MetroSchema } from './types/metro'
import './styles/App.css'

function App() {
  const [cities, setCities] = useState<City[]>([])
  const [citySlug, setCitySlug] = useState('saint-petersburg')
  const [schema, setSchema] = useState<MetroSchema | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [adminOpen, setAdminOpen] = useState(false)
  const [theme, setTheme] = useState<'light' | 'dark'>(() =>
    localStorage.getItem('theme') === 'dark' ? 'dark' : 'light'
  )

  const loadCities = (signal?: AbortSignal) => getCities(signal).then((loaded) => {
    setCities(loaded)
    return loaded
  })

  useEffect(() => {
    document.documentElement.dataset.theme = theme
    localStorage.setItem('theme', theme)
  }, [theme])

  useEffect(() => {
    const controller = new AbortController()
    loadCities(controller.signal)
      .then((loaded) => setCitySlug((current) =>
        loaded.some((city) => city.slug === current) ? current : (loaded[0]?.slug ?? '')
      ))
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
    await loadCities()
    setCitySlug(slug)
  }

  return (
    <main className="app">
      <section>
        <header className="map-header">
          <button
            type="button"
            className="theme-toggle"
            onClick={() => setTheme((current) => current === 'light' ? 'dark' : 'light')}
            aria-label={theme === 'light' ? 'Включить тёмную тему' : 'Включить светлую тему'}
            title={theme === 'light' ? 'Включить тёмную тему' : 'Включить светлую тему'}
          >
            {theme === 'light' ? (
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.7" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
                <path d="M20.985 12.486A9 9 0 0 1 11.514 3.015 9 9 0 1 0 20.985 12.486Z" />
              </svg>
            ) : (
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.7" strokeLinecap="round" aria-hidden="true">
                <circle cx="12" cy="12" r="4" />
                <path d="M12 2v2m0 16v2M4.93 4.93l1.42 1.42m11.3 11.3 1.42 1.42M2 12h2m16 0h2M4.93 19.07l1.42-1.42m11.3-11.3 1.42 1.42" />
              </svg>
            )}
          </button>
          <h1>Схема метро</h1>
          <CitySelector cities={cities} value={citySlug} onChange={setCitySlug} />
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