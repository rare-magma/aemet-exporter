package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"time"
)

type AemetWeatherRedirect struct {
	Datos string `json:"datos"`
}

type AemetWeather struct {
	Fint string  `json:"fint"`
	Ta   float64 `json:"ta"`
	Hr   float64 `json:"hr"`
	Pres float64 `json:"pres"`
	Vv   float64 `json:"vv"`
	Dv   float64 `json:"dv"`
	Vmax float64 `json:"vmax"`
	Prec float64 `json:"prec"`
	Tpr  float64 `json:"tpr"`
	Vis  float64 `json:"vis"`
	Inso float64 `json:"inso"`
	Nie  float64 `json:"nie"`
}

type AemetWeatherResponse struct {
	Datos []AemetWeather `json:"datos"`
}

type Config struct {
	AemetApiKey             string `json:"AemetApiKey"`
	AemetWeatherStationCode string `json:"AemetWeatherStationCode"`
	Bucket                  string `json:"Bucket"`
	InfluxDBHost            string `json:"InfluxDBHost"`
	InfluxDBApiToken        string `json:"InfluxDBApiToken"`
	Org                     string `json:"Org"`
}

const aemetApi = "https://opendata.aemet.es/opendata/api/observacion/convencional/datos/estacion/"

func main() {
	confFilePath := "aemet_exporter.json"
	confData, err := os.Open(confFilePath)
	if err != nil {
		log.Fatalln("Error reading config file: ", err)
	}
	defer confData.Close()
	var config Config
	err = json.NewDecoder(confData).Decode(&config)
	if err != nil {
		log.Fatalln("Error reading configuration: ", err)
	}
	if config.AemetApiKey == "" {
		log.Fatalln("AemetApiKey is required")
	}
	if config.AemetWeatherStationCode == "" {
		log.Fatalln("AemetWeatherStationCode is required")
	}
	if config.Bucket == "" {
		log.Fatalln("Bucket is required")
	}
	if config.InfluxDBHost == "" {
		log.Fatalln("InfluxDBHost is required")
	}
	if config.InfluxDBApiToken == "" {
		log.Fatalln("InfluxDBApiToken is required")
	}
	if config.Org == "" {
		log.Fatalln("Org is required")
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout: 30 * time.Second,
			}).DialContext,
			TLSHandshakeTimeout:   30 * time.Second,
			ResponseHeaderTimeout: 30 * time.Second,
		},
	}
	apiUrl := fmt.Sprintf(aemetApi+"%s", config.AemetWeatherStationCode)
	req, _ := http.NewRequest("GET", apiUrl, nil)
	req.Header.Set("api_key", config.AemetApiKey)
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalln("Error requesting data: ", err)
	}
	defer resp.Body.Close()
	redirectStatusOK := resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusBadRequest
	if !redirectStatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Fatalln("Error reading data: ", err)
		}
		log.Fatalln("Error fetching AEMET redirect URL: ", string(resp.Status), string(body))
	}
	aemetWeatherRedirectResp, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln("Error reading data: ", err)
	}
	var aemetWeatherRedirect AemetWeatherRedirect
	err = json.Unmarshal(aemetWeatherRedirectResp, &aemetWeatherRedirect)
	if err != nil {
		log.Fatalln("Error unmarshalling data: ", err)
	}

	aemetWeatherReq, _ := http.NewRequest("GET", aemetWeatherRedirect.Datos, nil)
	aemetWeatherResp, err := client.Do(aemetWeatherReq)
	if err != nil {
		log.Fatalln("Error requesting data: ", err)
	}
	defer resp.Body.Close()
	getStatusOK := resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusBadRequest
	if !getStatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Fatalln("Error reading data: ", err)
		}
		log.Fatalln("Error fetching AEMET data: ", string(resp.Status), string(body))
	}
	aemetWeatherData, err := io.ReadAll(aemetWeatherResp.Body)
	if err != nil {
		log.Fatalln("Error reading data: ", err)
	}
	var aemetWeather []AemetWeather
	err = json.Unmarshal(aemetWeatherData, &aemetWeather)
	if err != nil {
		log.Fatalln("Error unmarshalling data: ", err)
	}

	payload := bytes.Buffer{}
	for _, stat := range aemetWeather {
		timestamp, err := time.Parse(time.RFC3339, stat.Fint[:19]+"Z")
		if err != nil {
			log.Fatalln("Error parsing timestamp: ", err)
		}
		influxLine := fmt.Sprintf("aemet_weather_conditions,station=%s temperature=%.1f,humidity=%.1f,pressure=%.1f,windspeed=%.1f,winddirection=%.1f,windgust=%.1f,precipitation=%.1f,dewpoint=%.1f,visibility=%.1f,insolation=%.1f,snow=%.1f %v\n",
			config.AemetWeatherStationCode,
			stat.Ta,
			stat.Hr,
			stat.Pres,
			stat.Vv,
			stat.Dv,
			stat.Vmax,
			stat.Prec,
			stat.Tpr,
			stat.Vis,
			stat.Inso,
			stat.Nie,
			timestamp.Unix(),
		)
		payload.WriteString(influxLine)

	}

	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	w.Write(payload.Bytes())
	err = w.Close()
	if err != nil {
		log.Fatalln("Error compressing data: ", err)
	}
	url := fmt.Sprintf("https://%s/api/v2/write?precision=s&org=%s&bucket=%s", config.InfluxDBHost, config.Org, config.Bucket)
	post, _ := http.NewRequest("POST", url, &buf)
	post.Header.Set("Accept", "application/json")
	post.Header.Set("Authorization", "Token "+config.InfluxDBApiToken)
	post.Header.Set("Content-Encoding", "gzip")
	post.Header.Set("Content-Type", "text/plain; charset=utf-8")
	resp, err = client.Do(post)
	if err != nil {
		log.Fatalln("Error sending data: ", err)
	}
	defer resp.Body.Close()
	statusOK := resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices
	if !statusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Fatalln("Error reading data: ", err)
		}
		log.Fatalln("Error sending data: ", resp.Status, string(body))
	}
}
