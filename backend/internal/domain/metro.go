package domain

type City struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type Station struct {
	ID        int      `json:"id"`
	Name      string   `json:"name"`
	X         int      `json:"x"`
	Y         int      `json:"y"`
	CityID    int      `json:"-"`
	Year      int      `json:"year,omitempty"`
	Latitude  *float64 `json:"lat,omitempty"`
	Longitude *float64 `json:"lon,omitempty"`
}

type MetroLine struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Color    string `json:"color"`
	Stations []int  `json:"stations"`
	// Nil means connectivity has not been provided (including legacy schemas).
	Edges [][2]int `json:"edges"`
}

type MetroSchema struct {
	City     City        `json:"city"`
	Stations []Station   `json:"stations"`
	Lines    []MetroLine `json:"lines"`
}
