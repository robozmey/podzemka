import { useEffect, useState, type FormEvent } from 'react'
import { createCity, getCities, getMetroSchema, previewOverpassImport, type ImportPreview } from './api/metro'
import { MetroMap } from './components/MetroMap'
import type { City, MetroSchema } from './types/metro'
import './styles/App.css'
import { CitySelector } from './components/CitySelector'

function App() {
  const [cities, setCities] = useState<City[]>([])
  const [citySlug, setCitySlug] = useState('saint-petersburg')
  const [schema, setSchema] = useState<MetroSchema | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [cityName, setCityName] = useState('')
  const [newCitySlug, setNewCitySlug] = useState('')
  const [file, setFile] = useState<File | null>(null)
  const [preview, setPreview] = useState<ImportPreview | null>(null)
  const [importError, setImportError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)
  const [theme, setTheme] = useState<'light' | 'dark'>(() => {
    return localStorage.getItem('theme') === 'dark' ? 'dark' : 'light'
  })

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

  const handlePreview = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    if (!file) {
      setImportError('Выберите JSON-файл Overpass')
      return
    }
    setBusy(true)
    setImportError(null)
    setPreview(null)
    try {
      setPreview(await previewOverpassImport(file))
    } catch (reason) {
      setImportError(reason instanceof Error ? reason.message : 'Ошибка обработки Overpass JSON')
    } finally {
      setBusy(false)
    }
  }

  const handleCreate = async () => {
    if (!preview) return
    setBusy(true)
    setImportError(null)
    try {
      await createCity({ name: cityName, slug: newCitySlug }, preview.schema)
      const loaded = await loadCities()
      setCitySlug(newCitySlug)
      setCities(loaded)
      setPreview(null)
      setFile(null)
      setCityName('')
      setNewCitySlug('')
    } catch (reason) {
      setImportError(reason instanceof Error ? reason.message : 'Не удалось создать город')
    } finally {
      setBusy(false)
    }
  }

  const handleSlugInput = (value: string) => {
    setNewCitySlug(value.toLowerCase().trim().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, ''))
  }

  return (
    <main className="app">
      <section>
        <header className="map-header">
          <button
            type="button"
            className="theme-toggle"
            onClick={() => setTheme(theme === 'light' ? 'dark' : 'light')}
            aria-label={theme === 'light' ? 'Включить тёмную тему' : 'Включить светлую тему'}
            title={theme === 'light' ? 'Включить тёмную тему' : 'Включить светлую тему'}
          >
            {theme === 'light' ? (
              <svg
                width="18"
                height="18"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="1.7"
                strokeLinecap="round"
                strokeLinejoin="round"
              >
                <path d="M20.985 12.486A9 9 0 0 1 11.514 3.015 9 9 0 1 0 20.985 12.486Z" />
              </svg>
            ) : (
              <svg
                width="18"
                height="18"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="1.7"
                strokeLinecap="round"
              >
                <circle cx="12" cy="12" r="4" />
                <path d="M12 2v2m0 16v2M4.93 4.93l1.42 1.42m11.3 11.3 1.42 1.42M2 12h2m16 0h2M4.93 19.07l1.42-1.42m11.3-11.3 1.42-1.42" />
              </svg>
            )}

          </button>
          <h1>Схема метро</h1>
          <CitySelector
            cities={cities}
            value={citySlug}
            onChange={setCitySlug}
          />
        </header>

        <details className="city-import">
          <summary>Добавить город из Overpass JSON</summary>
          <form className="city-import__form" onSubmit={handlePreview}>
            <label>Название города<input required value={cityName} onChange={(event) => setCityName(event.target.value)} /></label>
            <label>Slug<input required pattern="[a-z0-9]+(-[a-z0-9]+)*" value={newCitySlug} onChange={(event) => handleSlugInput(event.target.value)} placeholder="kazan" /></label>
            <label>Overpass JSON<input required type="file" accept=".json,application/json" onChange={(event) => { setFile(event.target.files?.[0] ?? null); setPreview(null) }} /></label>
            <button type="submit" disabled={busy || !file}>{busy ? 'Обработка…' : 'Показать предпросмотр'}</button>
          </form>
          {importError && <p className="message message--error">{importError}</p>}
          {preview && (
            <div className="city-import__preview">
              <p>Найдено станций: {preview.stationsCount}; линий: {preview.linesCount}.</p>
              <p>Координаты рассчитаны автоматически. Перед созданием проверьте результат после выбора города.</p>
              <button type="button" onClick={handleCreate} disabled={busy || !cityName.trim() || !newCitySlug}>{busy ? 'Сохранение…' : 'Создать город и добавить на сайт'}</button>
            </div>
          )}
        </details>

        {error && <p className="message message--error">{error}</p>}
        {!schema && !error && <p className="message">Загрузка схемы...</p>}
        {schema && <MetroMap schema={schema} />}
      </section>
    </main>
  )
}

export default App
