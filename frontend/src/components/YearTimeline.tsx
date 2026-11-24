
import "./YearTimeline.css";
import type { MetroSchema } from "../types/metro";

export type MetroCitySlug = string;

export type TimelineMark = {
  year: number;
  label: string;
};

export const CITY_YEAR_RANGES: Record<
  string,
  { label: string; start: number; end: number }
> = {
  "moscow": {
    label: "Москва",
    start: 1935,
    end: 2026,
  },
  "saint-petersburg": {
    label: "Санкт-Петербург",
    start: 1955,
    end: 2026,
  },
  "krasnoyarsk": {
    label: "Красноярск",
    start: 1983, // дата начала планирования
    end: 2026,
  },
};

export const CITY_YEAR_MARKS: Record<
  string,
  TimelineMark[]
> = {
  "moscow": [
    { year: 1935, label: "Открытие метро" },
  ],
  "saint-petersburg": [
    { year: 1955, label: "Открытие метро" },
    { year: 1961, label: "Открыта 2-я линия" },
    { year: 1967, label: "Открыта 3-я линия" },
    { year: 1978, label: "Открыто «Девяткино»" },
    { year: 1985, label: "Открыта 4-я линия" },
    { year: 1997, label: "Открыта 5-я линия" },
    { year: 2008, label: "Продление на юг" },
    { year: 2018, label: "Продление на запад" },
  ],
  "krasnoyarsk": [
    { year: 1983, label: "Принято постановление о строительстве" },
  ],
};

export function getCityTimeline(schema: MetroSchema) {
  const city = schema.city.slug;
  const knownRange = Object.prototype.hasOwnProperty.call(CITY_YEAR_RANGES, city)
    ? CITY_YEAR_RANGES[city as MetroCitySlug]
    : undefined;
  const years = schema.stations.flatMap((station) =>
    typeof station.year === "number" && Number.isFinite(station.year) && station.year > 0
      ? [station.year]
      : [],
  );
  const currentYear = new Date().getFullYear();
  const start = years.length > 0 ? Math.min(...years) : 1900;
  const end = years.length > 0
    ? Math.max(currentYear, ...years)
    : currentYear;

  return {
    range: knownRange ?? { label: schema.city.name, start, end },
    marks: knownRange
      ? CITY_YEAR_MARKS[city as MetroCitySlug]
      : years.length > 0
        ? [{ year: start, label: "Первые найденные даты станций" }]
        : [{ year: start, label: "Исторический диапазон схемы" }],
    hasYears: true,
  };
}

type Props = {
  timeline: ReturnType<typeof getCityTimeline>;
  year: number;
  onYearChange: (year: number) => void;
};

export default function YearTimeline({
  timeline,
  year,
  onYearChange,
}: Props) {
  const { range } = timeline;
  const selectedYear = Math.max(
    range.start,
    Math.min(range.end, year)
  );

  const marks = timeline.marks.filter(
    (mark) =>
      mark.year >= range.start && mark.year <= range.end
  );

  return (
    <aside className="year-timeline">

      <div className="year-timeline__city">
        {range.label}
      </div>

      <div className="year-timeline__selected">
        {selectedYear}
      </div>

      <div className="year-timeline__scale">
        <input
          className="year-timeline__range"
          type="range"
          min={range.start}
          max={range.end}
          step={1}
          value={selectedYear}
          aria-label={`Выбор года для города ${range.label}`}
          onChange={(event) =>
            onYearChange(Number(event.target.value))
          }
        />

        {marks.map((mark) => {
          const position =
            ((mark.year - range.start) /
              (range.end - range.start || 1)) *
            100;

          return (
            <button
              key={`${mark.year}-${mark.label}`}
              type="button"
              className={`year-timeline__mark ${
                mark.year === range.start
                  ? "year-timeline__mark--start"
                  : ""
              }`}
              style={{ top: `${position}%` }}
              title={mark.label}
              onClick={() => onYearChange(mark.year)}
            >
              <span className="year-timeline__dot" />
              <span className="year-timeline__mark-text">
                <strong>{mark.year}</strong>
                <small>{mark.label}</small>
              </span>
            </button>
          );
        })}

        <span className="year-timeline__end">
          {range.end}
        </span>
      </div>

    </aside>
  );
}
