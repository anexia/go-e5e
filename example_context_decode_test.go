package e5e_test

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"go.anx.io/e5e/v2"
)

type WeatherData struct {
	PostalCode string `json:"postal_code"`
}
type AuthContext struct {
	AuthKey string `json:"auth_key"`
}

func FetchWeather(ctx context.Context, r e5e.Request[WeatherData, AuthContext]) (*e5e.Result, error) {
	// Now we could, for example, call an API with the key:
	req, _ := http.NewRequest("GET", "https://example.com/weather", http.NoBody)
	req.Header.Add("Authorization", "Bearer "+r.Context.Data.AuthKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching weather status: %w", err)
	}

	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	return &e5e.Result{Data: body}, nil
}

func Example_contextDecode() {
	e5e.AddHandlerFunc("FetchWeather", FetchWeather)
	e5e.Start(context.Background())
}
