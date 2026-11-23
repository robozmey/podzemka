
import { useEffect, useId, useRef, useState } from 'react'
import type { KeyboardEvent } from 'react'
import type { City } from '../types/metro'
import './CitySelector.css'

type Props = {
  cities: City[]
  value: string
  onChange: (slug: string) => void
}

export function CitySelector({ cities, value, onChange }: Props) {
  const [isOpen, setIsOpen] = useState(false)
  const rootRef = useRef<HTMLDivElement>(null)
  const buttonRef = useRef<HTMLButtonElement>(null)
  const optionRefs = useRef<(HTMLButtonElement | null)[]>([])
  const listId = useId()

  const selectedIndex = cities.findIndex(city => city.slug === value)
  const selectedCity = cities[selectedIndex]

  useEffect(() => {
    if (!isOpen) return

    const handleOutsideClick = (event: PointerEvent) => {
      if (!rootRef.current?.contains(event.target as Node)) {
        setIsOpen(false)
      }
    }

    document.addEventListener('pointerdown', handleOutsideClick)

    return () => {
      document.removeEventListener('pointerdown', handleOutsideClick)
    }
  }, [isOpen])

  useEffect(() => {
    if (isOpen && cities.length > 0) {
      optionRefs.current[Math.max(selectedIndex, 0)]?.focus()
    }
  }, [isOpen, selectedIndex, cities.length])

  const selectCity = (slug: string) => {
    onChange(slug)
    setIsOpen(false)
    buttonRef.current?.focus()
  }

  const handleKeyDown = (event: KeyboardEvent<HTMLDivElement>) => {
    if (event.key === 'Escape' && isOpen) {
      event.preventDefault()
      setIsOpen(false)
      buttonRef.current?.focus()
      return
    }

    if (event.key !== 'ArrowDown' && event.key !== 'ArrowUp') return

    event.preventDefault()

    if (!isOpen) {
      setIsOpen(true)
      return
    }

    const currentIndex = optionRefs.current.findIndex(
      option => option === document.activeElement
    )

    const direction = event.key === 'ArrowDown' ? 1 : -1
    const nextIndex =
      (Math.max(currentIndex, 0) + direction + cities.length) % cities.length

    optionRefs.current[nextIndex]?.focus()
  }

  return (
    <div
      className="city-selector"
      ref={rootRef}
      onKeyDown={handleKeyDown}
    >
      <span className="city-selector__label">Город</span>

      <button
        ref={buttonRef}
        type="button"
        className="city-selector__trigger"
        aria-expanded={isOpen}
        aria-controls={listId}
        aria-haspopup="true"
        disabled={cities.length === 0}
        onClick={() => setIsOpen(open => !open)}
      >
        <span>{selectedCity?.name ?? 'Выберите город'}</span>

        <svg
          className={`city-selector__arrow ${isOpen ? 'is-open' : ''}`}
          width="18"
          height="18"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="1.7"
          strokeLinecap="round"
          strokeLinejoin="round"
          aria-hidden="true"
        >
          <path d="m6 9 6 6 6-6" />
        </svg>
      </button>

      {isOpen && (
        <div id={listId} className="city-selector__menu">
          {cities.map((city, index) => (
            <button
              key={city.id}
              ref={element => { optionRefs.current[index] = element }}
              type="button"
              className={`city-selector__option ${
                city.slug === value ? 'is-selected' : ''
              }`}
              aria-current={city.slug === value ? 'true' : undefined}
              onClick={() => selectCity(city.slug)}
            >
              <span>{city.name}</span>

              {city.slug === value && (
                <svg
                  width="16"
                  height="16"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="1.8"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  aria-hidden="true"
                >
                  <path d="m5 12 4 4L19 6" />
                </svg>
              )}
            </button>
          ))}
        </div>
      )}
    </div>
  )
}
