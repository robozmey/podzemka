export type City = {
  id: number
  name: string
  slug: string
}

export type Station = {
  id: number
  name: string
  x: number
  y: number
}

export type MetroLine = {
  id: number
  name: string
  color: string
  stations: number[]
}

export type MetroSchema = {
  city: City
  stations: Station[]
  lines: MetroLine[]
}
