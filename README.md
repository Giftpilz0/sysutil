# Sysutil

I have been trying to learn Go, writing various small CLI tools.

## Usage of deviceapi command

```
curl 127.0.0.1:8080/volume
curl -X POST -d '{device:@DEFAULT_SINK@,level:0.5}' 127.0.0.1:8080/volume
curl 127.0.0.1:8080/volume/toggle/mute
curl 127.0.0.1:8080/network/ssid
curl 127.0.0.1:8080/network/signal
curl 127.0.0.1:8080/network/ip
curl 127.0.0.1:8080/network/toggle/wifi
```
