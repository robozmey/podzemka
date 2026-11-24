import { useEffect, useRef, useState, type FormEvent } from 'react'

import { createCity, getCatalogCities, previewCityByName, previewCityFile, verifyAdminToken, type CityPreview, type CatalogCity } from '../api/metro'

type AdminCityPanelProps = {
  open: boolean
  onClose: () => void
  onCityCreated: (slug: string) => Promise<void>
}

export function AdminCityPanel({ open, onClose, onCityCreated }: AdminCityPanelProps) {
  const dialogRef = useRef<HTMLDialogElement>(null)
  const [token, setToken] = useState('')
  const [tokenInput, setTokenInput] = useState('')
  const [authenticated, setAuthenticated] = useState(false)
  const [cityName, setCityName] = useState('')
  const [preview, setPreview] = useState<CityPreview | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)
  const [catalogCities, setCatalogCities] = useState<CatalogCity[]>([])
  const [importFile, setImportFile] = useState<File | null>(null)

  useEffect(() => {
    if (!open) return
    const controller = new AbortController()
    getCatalogCities(controller.signal)
      .then(setCatalogCities)
      .catch(() => setCatalogCities([]))
    return () => controller.abort()
  }, [open])

  useEffect(() => {
    const dialog = dialogRef.current
    if (!dialog) return
    if (open && !dialog.open) dialog.showModal()
    if (!open && dialog.open) dialog.close()
  }, [open])

  const handleLogin = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setBusy(true)
    setError(null)
    try {
      const candidate = tokenInput.trim()
      await verifyAdminToken(candidate)
      setToken(candidate)
      setAuthenticated(true)
      setTokenInput('')
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : 'Не удалось проверить доступ')
    } finally {
      setBusy(false)
    }
  }

  const handlePreview = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setBusy(true)
    setError(null)
    setPreview(null)
    try {
      setPreview(await previewCityByName(cityName.trim(), token))
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : 'Не удалось найти город')
    } finally {
      setBusy(false)
    }
  }

  const handleFilePreview = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    if (!importFile) return
    setBusy(true)
    setError(null)
    setPreview(null)
    try {
      setPreview(await previewCityFile(cityName.trim(), importFile, token))
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : 'Не удалось проверить JSON-файл')
    } finally {
      setBusy(false)
    }
  }

  const handleImportFileChange = async (file: File | null) => {
    setImportFile(file)
    setPreview(null)
    setError(null)
    if (!file || cityName.trim()) return
    try {
      const parsed = JSON.parse(await file.text()) as { city?: { name?: string } }
      const fileCityName = parsed.city?.name?.trim()
      if (fileCityName) setCityName(fileCityName)
    } catch {
      // The backend will show the detailed JSON parsing error after submit.
    }
  }

  const handleCreate = async () => {
    if (!preview) return
    setBusy(true)
    setError(null)
    try {
      await createCity(
        { name: preview.name, slug: preview.slug },
        preview.schema,
        token,
      )
      await onCityCreated(preview.slug)
      setCityName('')
      setPreview(null)
      closePanel()
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : 'Не удалось добавить город')
    } finally {
      setBusy(false)
    }
  }

  const handleLogout = () => {
    setAuthenticated(false)
    setToken('')
    setTokenInput('')
    setCityName('')
    setImportFile(null)
    setPreview(null)
    setError(null)
  }

  const closePanel = () => {
    if (dialogRef.current?.open) dialogRef.current.close()
    handleLogout()
    onClose()
  }

  const handleDialogClose = () => {
    handleLogout()
    onClose()
  }

  return (
    <dialog
      ref={dialogRef}
      className="admin-dialog"
      onClose={handleDialogClose}
      onMouseDown={(event) => {
        if (event.target === event.currentTarget) closePanel()
      }}
    >
      <header className="admin-dialog__header">
        <div>
          <p className="eyebrow">Панель управления</p>
          <h2>Добавить город</h2>
        </div>
        <button className="admin-dialog__close" type="button" aria-label="Закрыть" onClick={closePanel}>
          ×
        </button>
      </header>

      {!authenticated ? (
        <form className="admin-form" onSubmit={handleLogin}>
          <label>
            Токен администратора
            <input
              autoComplete="current-password"
              type="password"
              required
              value={tokenInput}
              onChange={(event) => setTokenInput(event.target.value)}
            />
          </label>
          <button type="submit" disabled={busy || !tokenInput.trim()}>
            {busy ? 'Проверка…' : 'Войти'}
          </button>
        </form>
      ) : (
        <>
          <div className="admin-dialog__toolbar">
            <span>Доступ администратора</span>
            <button type="button" className="admin-dialog__logout" onClick={handleLogout}>Выйти</button>
          </div>
          <form className="admin-form" onSubmit={handlePreview}>
            <label>
              Название города на русском
              <input
                autoFocus
                autoComplete="off"
                list="recommended-cities"
                required
                maxLength={100}
                value={cityName}
                onChange={(event) => {
                  setCityName(event.target.value)
                  setPreview(null)
                }}
                placeholder="Казань"
              />
              <datalist id="recommended-cities">
                {catalogCities.map((city) => (
                  <option key={`${city.country}-${city.name}`} value={city.name}>
                    {city.country}
                  </option>
                ))}
              </datalist>
            </label>
            <button type="submit" disabled={busy || !cityName.trim()}>
              {busy ? 'Поиск в OpenStreetMap… это может занять до 2,5 минут' : 'Найти станции'}
            </button>
          </form>
          <form className="admin-form" onSubmit={handleFilePreview}>
            <label>
              Или импортировать JSON-файл
              <input
                type="file"
                accept=".json,application/json"
                onChange={(event) => {
                  void handleImportFileChange(event.target.files?.[0] ?? null)
                }}
              />
            </label>
            <button type="submit" disabled={busy || !importFile}>
              {busy ? 'Проверка файла…' : 'Проверить JSON-файл'}
            </button>
          </form>
          {preview && (
            <div className="admin-preview" aria-live="polite">
              <p>{preview.name}: найдено станций {preview.stationsCount}, линий {preview.linesCount}.</p>
              <p className="admin-preview__slug">Адресный slug: {preview.slug}</p>
              <button type="button" disabled={busy} onClick={handleCreate}>
                {busy ? 'Сохранение…' : 'Добавить на сайт'}
              </button>
            </div>
          )}
        </>
      )}
      {error && <p className="message message--error" role="alert">{error}</p>}
    </dialog>
  )
}
