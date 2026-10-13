import type { City, MetroSchema } from '../types/metro'

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
