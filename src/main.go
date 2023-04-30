package main

import (
	"fmt"
	"github.com/cheggaaa/pb/v3"
	"io/ioutil"
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

func downloadTile(z, x, y int, downloadedSize *int64, bar *pb.ProgressBar, wg *sync.WaitGroup) {
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

	data, err := ioutil.ReadAll(resp.Body)
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
	bar.Increment()

	if *downloadedSize > maxStorage {
		bar.Finish()
		os.Exit(0)
	}
}

type Triplet struct {
	X, Y, Z int
}

func main() {
	minZoom, maxZoom := 0, 17

	minLat, maxLat, minLng, maxLng := 47.1005, 47.1247, -2.1119, -2.0675

	var downloadedSize int64

	bar := pb.StartNew((maxZoom - minZoom + 1) * int((maxLat-minLat)*111111) * int((maxLng-minLng)*111111))

	var wg sync.WaitGroup

	visited := make(map[Triplet]bool)
	for zoom := minZoom; zoom <= maxZoom; zoom++ {
		for lat := minLat; lat <= maxLat; lat += 1.0 / (111111 * float64(uint(1)<<uint(zoom))) {
			y := latToTileY(lat, zoom)
			for lng := minLng; lng <= maxLng; lng += 1.0 / (111111 * float64(uint(1)<<uint(zoom))) {
				x := lngToTileX(lng, zoom)
				coordinates := Triplet{x, y, zoom}
				if visited[coordinates] {
					continue
				}
				visited[coordinates] = true
				wg.Add(1)
				go downloadTile(zoom, x, y, &downloadedSize, bar, &wg)
			}
		}
	}

	wg.Wait()
	bar.Finish()
}

func lngToTileX(lng float64, zoom int) int {
	return int((lng + 180.0) / 360.0 * float64(uint(1)<<uint(zoom)))
}

func latToTileY(lat float64, zoom int) int {
	latRad := lat * (3.14159265359 / 180.0)
	return int((1.0 - math.Log(math.Tan(latRad)+(1.0/math.Cos(latRad)))/math.Pi) / 2.0 * float64(uint(1)<<uint(zoom)))
}
