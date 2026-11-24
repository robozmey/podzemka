export type City = {
  id: number
  name: string
  slug: string
}

export type Station = {
  id: number
  name: string
  year?: number
  x: number
  y: number
  lat?: number
  lon?: number
}

export type MetroLine = {
  id: number
  name: string
  color: string
  stations: number[]
  edges?: [number, number][] | null
}

export type MetroSchema = {
  city: City
  stations: Station[]
  lines: MetroLine[]
}
