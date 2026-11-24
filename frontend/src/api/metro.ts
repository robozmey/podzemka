import type { City, MetroSchema, Station, MetroLine } from '../types/metro'

export type CatalogCity = {
  name: string
  country: string
}

export type CityPreview = {
  name: string
  slug: string
  stationsCount: number
  linesCount: number
  schema: Pick<MetroSchema, 'stations' | 'lines'> & {
    stations: Station[]
    lines: MetroLine[]
  }
}

async function throwApiError(response: Response, fallback: string): Promise<never> {
  const body = await response.json().catch(() => null) as { error?: string } | null
  throw new Error(body?.error ?? fallback)
}

export async function verifyAdminToken(token: string, signal?: AbortSignal): Promise<void> {
  const response = await fetch('/api/admin/session', {
    headers: { Authorization: `Bearer ${token}` },
    signal,
  })
  if (!response.ok) {
    await throwApiError(response, 'Нет доступа к администрированию')
  }
}

export async function previewCityByName(
  name: string,
  token: string,
  signal?: AbortSignal,
): Promise<CityPreview> {
  const response = await fetch('/api/admin/cities/preview', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify({ name }),
    signal,
  })
  if (!response.ok) {
    await throwApiError(response, 'Не удалось найти станции города')
  }
  return response.json() as Promise<CityPreview>
}

export async function previewCityFile(
  name: string,
  file: File,
  token: string,
  signal?: AbortSignal,
): Promise<CityPreview> {
  const formData = new FormData()
  formData.append('name', name)
  formData.append('file', file)
  const timeout = new AbortController()
  const timeoutId = window.setTimeout(() => timeout.abort(), 60_000)
  if (signal) signal.addEventListener('abort', () => timeout.abort(), { once: true })
  try {
    const response = await fetch('/api/cities/preview-import', {
      method: 'POST',
      headers: { Authorization: `Bearer ${token}` },
      body: formData,
      signal: timeout.signal,
    })
    if (!response.ok) {
      await throwApiError(response, 'Не удалось проверить JSON-файл')
    }
    return response.json() as Promise<CityPreview>
  } finally {
    window.clearTimeout(timeoutId)
  }
}

export async function createCity(
  city: { name: string; slug: string },
  schema: CityPreview['schema'],
  token: string,
  signal?: AbortSignal,
): Promise<void> {
  const response = await fetch('/api/cities', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify({ ...city, ...schema }),
    signal,
  })
  if (!response.ok) {
    await throwApiError(response, 'Не удалось создать город')
  }
}

export async function getMetroSchema(
  citySlug: string,
  signal?: AbortSignal,
): Promise<MetroSchema> {
  const response = await fetch(`/api/cities/${citySlug}/metro/schema`, { signal })

  if (!response.ok) {
    throw new Error('Не удалось загрузить схему метро')
  }

  return response.json() as Promise<MetroSchema>
}

export async function getCities(signal?: AbortSignal): Promise<City[]> {
  const response = await fetch('/api/cities', { signal })

  if (!response.ok) {
    throw new Error('Не удалось загрузить список городов')
  }

  return response.json() as Promise<City[]>
}

export async function getCatalogCities(signal?: AbortSignal): Promise<CatalogCity[]> {
  const response = await fetch('/api/catalog/cities', { signal })

  if (!response.ok) {
    throw new Error('Не удалось загрузить каталог городов')
  }

  return response.json() as Promise<CatalogCity[]>
}
