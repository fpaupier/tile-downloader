package main

import (
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
	apiURL     = "http://a.basemaps.cartocdn.com/light_all/%d/%d/%d.png"
	outputDir  = "data/Pornic"
	maxStorage = 10 * (1 << 30) // 10 GB
	tileSize   = 8 * (1 << 10)  // 8 KB
)

func downloadTile(z, x, y int, downloadedSize *int64, wg *sync.WaitGroup) {
	defer wg.Done()

	url := fmt.Sprintf(apiURL, z, x, y)
	resp, err := http.Get(url)
	if err != nil {
		log.Printf("Failed to download tile %d/%d/%d: %v\n", z, x, y, err)
		return
	}
	defer resp.Body.Close()

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

func main() {
	minZoom, maxZoom := 17, 22

	minLat, maxLat, minLng, maxLng := 47.1005, 47.1247, -2.1119, -2.0675

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
