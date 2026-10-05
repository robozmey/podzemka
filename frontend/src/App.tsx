import { useEffect, useState } from 'react'
import { getMetroSchema } from './api/metro'
import { MetroMap } from './components/MetroMap'
import type { MetroSchema } from './types/metro'
import './styles/App.css'

const citySlug = 'saint-petersburg'

function App() {
  const [schema, setSchema] = useState<MetroSchema | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    const controller = new AbortController()

    getMetroSchema(citySlug, controller.signal)
      .then(setSchema)
      .catch((reason: unknown) => {
        if (reason instanceof Error && reason.name !== 'AbortError') {
          setError(reason.message)
        }
      })

    return () => controller.abort()
  }, [])

  return (
    <main className="app">
      <section>
        <h1>{schema ? `Схема метро: ${schema.city.name}` : 'Схема метро'}</h1>
        {error && <p className="message message--error">{error}</p>}
        {!schema && !error && <p className="message">Загрузка схемы...</p>}
        {schema && <MetroMap schema={schema} />}
      </section>
    </main>
  )
}

export default App
