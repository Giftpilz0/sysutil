# Sysutil

I have been trying to learn Go, writing various small CLI tools.

# Usage of deviceapi command

## The following endpoints are provided:

GET /network → Returns the JSON output from GetNetworkDevices.
GET /battery → Returns the JSON output from GetBatteryStatus.
GET /audio/outputs → Returns the JSON output from GetVolumeInfo.
GET /audio/inputs → Returns the JSON output from GetInputInfo.
POST /audio/actions → Accepts JSON input for ProcessAudioActions and returns a status.

### POST example:

```
curl -X POST -d '[{"device":"alsa_output.usb-Plantronics_Plantronics_Blackwire_5220_Series_02FCAAAB685740D3A43CCE7C8DF13E03-00.analog-stereo","adjust":50,"muted":false,"default":true,"type":"sink"}]' 127.0.0.1:8090/audio/actions
```
