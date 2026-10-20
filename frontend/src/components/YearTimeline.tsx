
import "./YearTimeline.css";

export type MetroCitySlug =
  | "moscow"
  | "saint-petersburg"
  | "krasnoyarsk";

export type TimelineMark = {
  year: number;
  label: string;
};

export const CITY_YEAR_RANGES: Record<
  MetroCitySlug,
  { label: string; start: number; end: number }
> = {
  "moscow": {
    label: "Москва",
    start: 1935,
    end: 2024,
  },
  "saint-petersburg": {
    label: "Санкт-Петербург",
    start: 1955,
    end: 2024,
  },
  "krasnoyarsk": {
    label: "Красноярск",
    start: 1983, // дата начала планирования
    end: 2024,
  },
};

export const CITY_YEAR_MARKS: Record<
  MetroCitySlug,
  TimelineMark[]
> = {
  "moscow": [
    { year: 1935, label: "Открытие метро" },
  ],
  "saint-petersburg": [
    { year: 1955, label: "Открытие метро" },
  ],
  "krasnoyarsk": [
    { year: 1983, label: "Принято постановление о строительстве" },
  ],
};

type Props = {
  city: MetroCitySlug;
  year: number;
  onYearChange: (year: number) => void;
};

export default function YearTimeline({
  city,
  year,
  onYearChange,
}: Props) {
  const range = CITY_YEAR_RANGES[city];
  const selectedYear = Math.max(
    range.start,
    Math.min(range.end, year)
  );

  const marks = CITY_YEAR_MARKS[city].filter(
    (mark) =>
      mark.year >= range.start && mark.year <= range.end
  );

  return (
    <aside className="year-timeline">
      <div className="year-timeline__heading">
        ВЫБОР ГОДА
      </div>

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
              (range.end - range.start)) *
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
