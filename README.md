# Tile downloader

Download tiles from a TMS (_Tile Media Server_)

List of TMS layers providers: https://community.tibco.com/s/article/GeoAnalytics-Resources-WMS-and-TMS-Layers
this project uses this layer TMS URL: http://a.basemaps.cartocdn.com/light_all/{z}/{x}/{y}.png


## Get started 

Clone this project 
```shell
git clone git@github.com:fpaupier/tile-downloader.git
```

Update your location of interest in `src/main.go`, see:
`minLat, maxLat, minLng, maxLng := 47.1005, 47.1247, -2.1119, -2.0675` (Pornic in France for example)


Build with go
```shell
go build src/main.go
```

Run the build object
```shell
./main
```