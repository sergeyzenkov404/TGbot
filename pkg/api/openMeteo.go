package api

const (
	UrlMeteo = "https://api.open-meteo.com/v1/"
)

type MeteoResponse struct {
	TimeZone     string            `json:"timezone"`
	Current      CurrentValueMeteo `json:"current"`
	CurrentUnits CurrentUnitsMeteo `json:"current_units"`
}

type CurrentValueMeteo struct {
	Time        string  `json:"time"`
	Temperature float64 `json:"temperature_2m"`
	Wind        float64 `json:"wind_speed_10m"`
}
type CurrentUnitsMeteo struct {
	Time        string `json:"time"`
	Temperature string `json:"temperature_2m"`
	Wind        string `json:"wind_speed_10m"`
}
