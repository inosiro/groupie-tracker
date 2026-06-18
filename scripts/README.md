download a single csv file from [world cities](https://simplemaps.com/data/world-cities)
that contains 50000 cities, below is a sample of rows:

```
"city","city_ascii","lat","lng","country","iso2","iso3","admin_name","capital","population","id"
"Tokyo","Tokyo","35.6850","139.7514","Japan","JP","JPN","Tōkyō","primary","37785000","1392685764"
"Jakarta","Jakarta","-6.1753","106.8269","Indonesia","ID","IDN","Jakarta","primary","33756000","1360771077"
"Delhi","Delhi","28.6600","77.2300","India","IN","IND","Delhi","admin","32226000","1356872604"
"Chongqing","Chongqing","29.5500","106.5069","China","CN","CHN","Chongqing","admin","32054159","1156936556"
"Guangzhou","Guangzhou","23.1300","113.2600","China","CN","CHN","Guangdong","admin","26940000","1156237133"
"Mumbai","Mumbai","19.0758","72.8775","India","IN","IND","Mahārāshtra","admin","24973000","1356226629"
"Manila","Manila","14.5958","120.9772","Philippines","PH","PHL","Manila","primary","24922000","1608618140"
"Shanghai","Shanghai","31.2325","121.4692","China","CN","CHN","Shanghai","admin","24870895","1156073548"
"São Paulo","Sao Paulo","-23.5504","-46.6339","Brazil","BR","BRA","São Paulo","admin","23086000","1076532519"
```

because the file is >5MB, we are going to filter the file based on all concert locations to
use it as a geolocation resolver string -> (lng, lat)

create a list of concert locations that we want:
save `https://groupietrackers.herokuapp.com/api/locations` to locations.json
`jq -r '.index[].locations[]' locations.json |sort -u >locations.txt`