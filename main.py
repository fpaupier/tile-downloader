import os
import requests
from tqdm import tqdm
from concurrent.futures import ProcessPoolExecutor, as_completed

API_URL = "http://a.basemaps.cartocdn.com/light_all/{z}/{x}/{y}.png"
OUTPUT_DIR = "France"
MAX_STORAGE = 10 * (1024 ** 3)  # 10 GB
TILE_SIZE = 8 * (1024)  # 8 KB


# Bounding boxes and coordinates
WORLD_BBOX = (-180, -90, 180, 90)
FRANCE_BBOX = (-5.1406, 41.3337, 9.5593, 51.0890)
PAYS_BASQUE_BBOX = (-1.898, 43.139, -1.166, 43.582)
BAYONNE_COORD = (-1.4748, 43.4832)
BIARRITZ_COORD = (-1.5586, 43.4715)
ANGLET_COORD = (-1.5177, 43.4782)

# Zoom levels for different regions
WORLD_ZOOM = 1
FRANCE_ZOOM = 3
PAYS_BASQUE_ZOOM = 5
CITY_ZOOM = 22


def tile_bounds(z, x, y):
    n = 2 ** z
    lon1 = x / n * 360.0 - 180.0
    lat1 = y / n * 360.0 - 180.0
    lon2 = (x + 1) / n * 360.0 - 180.0
    lat2 = (y + 1) / n * 360.0 - 180.0
    return lon1, lat1, lon2, lat2


def in_bbox(bbox, lon, lat):
    return bbox[0] <= lon <= bbox[2] and bbox[1] <= lat <= bbox[3]


def download_tile(z, x, y):
    url = API_URL.format(z=z, x=x, y=y)
    response = requests.get(url)
    if response.status_code == 200:
        path = os.path.join(OUTPUT_DIR, str(z), str(x))
        os.makedirs(path, exist_ok=True)
        with open(os.path.join(path, f"{y}.png"), "wb") as f:
            f.write(response.content)
        return True
    return False


def main():
    downloaded_size = 0
    max_zoom = 22
    total_tiles = sum(2 ** (2 * z) for z in range(max_zoom))
    progress_bar = tqdm(total=total_tiles, unit="tile")

    with ProcessPoolExecutor() as executor:
        futures = []
        for z in range(0, max_zoom):
            for x in range(2 ** z):
                for y in range(2 ** z):
                    lon1, lat1, lon2, lat2 = tile_bounds(z, x, y)
                    if in_bbox(FRANCE_BBOX, lon1, lat1) or in_bbox(FRANCE_BBOX, lon2, lat2):
                        futures.append(executor.submit(download_tile, z, x, y))

        for future in as_completed(futures):
            if future.result():
                downloaded_size += TILE_SIZE
                if downloaded_size > MAX_STORAGE:
                    progress_bar.close()
                    return
            progress_bar.update(1)

    progress_bar.close()


if __name__ == "__main__":
    main()
