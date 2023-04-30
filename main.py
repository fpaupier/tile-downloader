import os
import requests
from tqdm import tqdm
from concurrent.futures import ThreadPoolExecutor, as_completed

API_URL = "http://a.basemaps.cartocdn.com/light_all/{z}/{x}/{y}.png"
OUTPUT_DIR = "data/France"
MAX_STORAGE = 10 * (1024 ** 3)  # 10 GB
TILE_SIZE = 8 * 1024  # 8 KB

# Bounding boxes and zoom levels for the specified areas
WORLD = {"bbox": (-180, -90, 180, 90), "zoom": 1}
FRANCE = {"bbox": (-5.1406, 41.3337, 9.5593, 51.0890), "zoom": 2}
PAYS_BASQUE = {"bbox": (-1.9, 43.0, -1.2, 43.5), "zoom": 4}
CITIES = [
    {"name": "Bayonne", "bbox": (-1.5106, 43.4674, -1.4404, 43.5173), "zoom": 22},
    {"name": "Biarritz", "bbox": (-1.5908, 43.4437, -1.5236, 43.5072), "zoom": 22},
    {"name": "Anglet", "bbox": (-1.5531, 43.4674, -1.4823, 43.5263), "zoom": 22},
]


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


def area_zoom_level(area, z, x, y):
    lon1, lat1, lon2, lat2 = tile_bounds(z, x, y)
    if in_bbox(area["bbox"], lon1, lat1) or in_bbox(area["bbox"], lon2, lat2):
        return z <= area["zoom"]
    return False


def main():
    downloaded_size = 0
    max_zoom = 5
    total_tiles = sum(2 ** (2 * z) for z in range(max_zoom))
    progress_bar = tqdm(total=total_tiles, unit="tile")

    for z in range(0, max_zoom):
        for x in range(2 ** z):
            for y in range(2 ** z):
                # Whole world zoom level 1
                if area_zoom_level(WORLD, z, x, y):
                    download_tile(z, x, y)

                # France zoom level 2
                if area_zoom_level(FRANCE, z, x, y):
                    download_tile(z, x, y)

                # French Pays Basque zoom level 4
                if area_zoom_level(PAYS_BASQUE, z, x, y):
                    download_tile(z, x, y)

                # Bayonne, Biarritz, and Anglet max zoom level
                for city in CITIES:
                    if area_zoom_level(city, z, x, y):
                        download_tile(z, x, y)

            downloaded_size += TILE_SIZE
            if downloaded_size > MAX_STORAGE:
                progress_bar.close()
                return
        progress_bar.update(1)
    progress_bar.close()


if __name__ == "__main__":
    main()
