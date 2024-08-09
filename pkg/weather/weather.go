package weather

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"runtime"
	"time"
)

type (
	cachedWeather struct {
		cachePath string
		token     string
		location  string
	}

	geoResp struct {
		Name  string  `json:"name"`
		State string  `json:"state"`
		Lat   float64 `json:"lat"`
		Lon   float64 `json:"lon"`
	}

	weatherResp struct {
		Weather []struct {
			ID          int    `json:"id"`
			Main        string `json:"main"`
			Description string `json:"description"`
		} `json:"weather"`
		Main struct {
			Temp      float64 `json:"temp"`
			FeelsLike float64 `json:"feels_like"`
		} `json:"main"`
		Sys struct {
			Sunrise int64 `json:"sunrise"`
			Sunset  int64 `json:"sunset"`
		} `json:"sys"`
	}

	cache struct {
		Location       *geoResp     `json:"location"`
		Weather        *weatherResp `json:"weather"`
		CachedLocation string       `json:"cached_location"`
		CacheExpiresAt *time.Time   `json:"cache_expires_at"`
	}

	output struct {
		Text       string `json:"text"`
		Alt        string `json:"alt"`
		Tooltip    string `json:"tooltip"`
		Class      string `json:"class"`
		Percentage string `json:"percentage"`
	}
)

func New(cachePath, location, token string) *cachedWeather {
	if cachePath == "" {
		_, filename, _, _ := runtime.Caller(0)
		cachePath = path.Join(path.Dir(filename), "weather-cache.json")
	}

	return &cachedWeather{
		cachePath: cachePath,
		token:     token,
		location:  location,
	}
}

func (cw *cachedWeather) Get() error {
	var c cache

	raw, err := os.ReadFile(cw.cachePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// ok
		} else {
			return err
		}
	} else {
		err = json.Unmarshal(raw, &c)
		if err != nil {
			return err
		}

		if c.CachedLocation == cw.location && c.CacheExpiresAt != nil && time.Until(*c.CacheExpiresAt) > 0 {
			return cw.print(c)
		}
	}

	if c.Location == nil {
		geo, err := queryGeo(cw.location, cw.token)
		if err != nil {
			return err
		}

		c.Location = geo
	}

	c.Weather, err = queryForecast(c.Location.Lat, c.Location.Lon, cw.token)
	if err != nil {
		return err
	}

	expiresAt := time.Now().Add(5 * time.Minute)
	c.CacheExpiresAt = &expiresAt
	c.CachedLocation = cw.location

	raw, err = json.Marshal(c)
	if err != nil {
		return err
	}

	err = os.WriteFile(cw.cachePath, raw, 0777) // nolint:gosec
	if err != nil {
		return err
	}

	return cw.print(c)
}

func (cw *cachedWeather) print(c cache) error {
	o := output{
		Text:       fmt.Sprintf("%.2f°C", c.Weather.Main.Temp),
		Alt:        icon(c.Weather.Weather[0].ID, c.Weather.isNight()),
		Tooltip:    fmt.Sprintf("%s, %s, %s", c.Location.Name, c.Location.State, c.Weather.Weather[0].Description),
		Class:      "",
		Percentage: "",
	}

	raw, err := json.Marshal(o)
	if err != nil {
		return err
	}

	fmt.Println(string(raw))

	return nil
}

func queryForecast(lat, lon float64, token string) (*weatherResp, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	values := url.Values{}
	values.Add("lat", fmt.Sprintf("%f", lat))
	values.Add("lon", fmt.Sprintf("%f", lon))
	values.Add("units", "metric")
	values.Add("appid", token)

	u, err := url.Parse("https://api.openweathermap.org/data/2.5/weather")
	if err != nil {
		return nil, err
	}

	u.RawQuery = values.Encode()

	r, err := req[weatherResp](ctx, u.String())
	if err != nil {
		return nil, err
	}

	if len(r.Weather) == 0 {
		return nil, fmt.Errorf("weather not contained")
	}

	return &r, nil

}

func queryGeo(location, token string) (*geoResp, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	values := url.Values{}
	values.Add("q", location)
	values.Add("limit", "1")
	values.Add("appid", token)

	u, err := url.Parse("https://api.openweathermap.org/geo/1.0/direct")
	if err != nil {
		return nil, err
	}

	u.RawQuery = values.Encode()

	r, err := req[[]geoResp](ctx, u.String())
	if err != nil {
		return nil, err
	}

	if len(r) == 0 || r[0].Lat == 0 || r[0].Lon == 0 {
		return nil, fmt.Errorf("invalid geo response: %v", r)
	}

	return &r[0], nil
}

func req[E any](ctx context.Context, url string) (E, error) {
	var zero E

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return zero, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return zero, err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return zero, err
	}

	if resp.StatusCode != http.StatusOK {
		return zero, fmt.Errorf("status code was not 200: %s", raw)
	}

	var r E
	err = json.Unmarshal(raw, &r)
	if err != nil {
		return zero, err
	}

	return r, nil
}

func icon(code int, night bool) string {
	// return ""
	switch {
	case code >= 200 && code < 300: // thunderstorm
		return ""
	case code >= 300 && code < 400: // drizzle
		return ""
	case code >= 500 && code < 600: // rain
		switch {
		case code < 505 && night:
			return ""
		case code < 505:
			return ""
		case code == 505:
			return ""
		default:
			return ""
		}
	case code >= 600 && code < 700: // snow
		return ""
	case code >= 700 && code < 800: // atmosphere
		switch {
		case code < 762:
			return ""
		case code == 762:
			return ""
		case code == 771:
			return ""
		case code == 781:
			return ""
		default:
			return ""
		}
	case code >= 800 && code < 900: // clouds
		switch {
		case code == 800 && night:
			return ""
		case code == 800:
			return ""
		case code == 801 && night:
			return ""
		case code == 801:
			return ""
		case code == 802:
			return ""
		case code == 802:
			return ""
		default:
			return ""
		}
	default:
		return ""
	}
}

func (w weatherResp) isNight() bool {
	now := time.Now()
	return now.After(time.Unix(w.Sys.Sunset, 0)) &&
		now.Before(time.Unix(w.Sys.Sunrise, 0))
}
