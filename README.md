# Sysutil

I have been trying to learn Go, writing various small CLI tools.

## Usage of deviceapi command

```
curl 127.0.0.1:8080/volume
curl 127.0.0.1:8080/volume | jq '.[] | select(.device == "@DEFAULT_SINK@") | .level'
curl -X POST -d '[{"device":"@DEFAULT_SINK@","adjust":0.1,"muted":false}]' 127.0.0.1:8080/volume
curl -X POST -d '[{"device":"@DEFAULT_SINK@","adjust":5%+,"muted":false}]' 127.0.0.1:8080/volume

curl 127.0.0.1:8080/network
curl 127.0.0.1:8080/network/toggle/wifi

curl 127.0.0.1:8080/battery/
```
