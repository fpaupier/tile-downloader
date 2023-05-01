package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"
)

const (
	maxStorage = 10 * (1 << 30) // 10 GB
	tileSize   = 8 * (1 << 10)  // 8 KB - approx size of each 256x256px tile
)

func downloadTile(z, x, y int, downloadedSize *int64, wg *sync.WaitGroup) {
	defer wg.Done()

	apiURL := os.Getenv("TMS_SERVER_URL")
	outputDir := os.Getenv("OUTPUT_DIR")

	url := fmt.Sprintf(apiURL, z, x, y)
	resp, err := http.Get(url)
	if err != nil {
		log.Printf("Failed to download tile %d/%d/%d: %v\n", z, x, y, err)
		return
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Printf("Failed to close body: %v\n", err)
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		log.Printf("Failed to download tile %d/%d/%d: status code %d\n", z, x, y, resp.StatusCode)
		return
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read tile %d/%d/%d: %v\n", z, x, y, err)
		return
	}

	tilePath := filepath.Join(outputDir, strconv.Itoa(z), strconv.Itoa(x))
	err = os.MkdirAll(tilePath, 0755)
	if err != nil {
		log.Printf("Failed to create directory %s: %v\n", tilePath, err)
		return
	}
	fPath := filepath.Join(tilePath, fmt.Sprintf("%d.png", y))

	err = os.WriteFile(fPath, data, 0644)
	if err != nil {
		log.Printf("Failed to write tile %d/%d/%d: %v\n", z, x, y, err)
		return
	}
	log.Printf("Tile written to: %s\n", fPath)

	*downloadedSize += tileSize

	if *downloadedSize > maxStorage {
		os.Exit(0)
	}
}

func parseZoom() (int, int, error) {
	minZoomEnv, maxZoomEnv := os.Getenv("MIN_ZOOM"), os.Getenv("MAX_ZOOM")
	// Parse the string value as integer
	minZoom, err := strconv.Atoi(minZoomEnv)
	if err != nil {
		fmt.Printf("Error parsing minZoom %s as an integer: %v\n", minZoomEnv, err)
		return 0, 0, errors.New("failed to parse minZoom as int")
	}

	maxZoom, err := strconv.Atoi(maxZoomEnv)
	if err != nil {
		fmt.Printf("Error parsing maxZoom %s as an integer: %v\n", maxZoomEnv, err)
		return 0, 0, errors.New("failed to parse maxZoom as int")
	}
	return minZoom, maxZoom, nil
}

func parseCoordinates() (float64, float64, float64, float64, error) {
	minLat, maxLat := os.Getenv("MIN_LAT"), os.Getenv("MAX_LAT")
	minLng, maxLng := os.Getenv("MIN_LNG"), os.Getenv("MAX_LNG")

	// Parse the string value as float64
	minLatFloat, err := strconv.ParseFloat(minLat, 64)
	if err != nil {
		fmt.Printf("Error parsing %s as float64: %v\n", minLat, err)
		return 0, 0, 0, 0, err
	}
	maxLatFloat, err := strconv.ParseFloat(maxLat, 64)
	if err != nil {
		fmt.Printf("Error parsing %s as float64: %v\n", maxLat, err)
		return 0, 0, 0, 0, err
	}

	minLngFloat, err := strconv.ParseFloat(minLng, 64)
	if err != nil {
		fmt.Printf("Error parsing %s as float64: %v\n", minLng, err)
		return 0, 0, 0, 0, err
	}

	maxLngFloat, err := strconv.ParseFloat(maxLng, 64)
	if err != nil {
		fmt.Printf("Error parsing %s as float64: %v\n", maxLng, err)
		return 0, 0, 0, 0, err
	}

	return minLatFloat, maxLatFloat, minLngFloat, maxLngFloat, nil
}

func main() {
	minZoom, maxZoom, err := parseZoom()
	if err != nil {
		fmt.Printf("Error parsing zoom: %v\n", err)
		os.Exit(-1)
	}

	minLat, maxLat, minLng, maxLng, err := parseCoordinates()
	if err != nil {
		fmt.Printf("Error parsing coordinates: %v\n", err)
		os.Exit(-1)
	}

	var downloadedSize int64

	var wg sync.WaitGroup

	// Limit the number of concurrent downloads
	concurrencyLimit := 50
	sem := make(chan struct{}, concurrencyLimit)

	//visited := make(map[Triplet]bool)
	for zoom := minZoom; zoom <= maxZoom; zoom++ {
		minTileX, minTileY := latLngToTileXY(minLat, minLng, zoom)
		maxTileX, maxTileY := latLngToTileXY(maxLat, maxLng, zoom)
		if maxTileY < minTileY { // Handle use case where transformation shift min/max
			minTileY, maxTileY = maxTileY, minTileY
		}

		for x := minTileX; x <= maxTileX; x++ {
			for y := minTileY; y <= maxTileY; y++ {
				wg.Add(1)
				// Acquire a semaphore
				sem <- struct{}{}
				go func(zoom, x, y int) {
					downloadTile(zoom, x, y, &downloadedSize, &wg)
					// Release the semaphore
					<-sem
				}(zoom, x, y)
			}
		}
	}

	wg.Wait()
}

func latLngToTileXY(lat, lng float64, zoom int) (int, int) {
	x := int(math.Floor((lng + 180.0) / 360.0 * math.Pow(2, float64(zoom))))
	y := int(math.Floor((1.0 - math.Log(math.Tan(lat*math.Pi/180.0)+1.0/math.Cos(lat*math.Pi/180.0))/math.Pi) / 2.0 * math.Pow(2, float64(zoom))))
	return x, y
}
