# PanGo

**PanGo** is a pan-tilt photography controller for automated grid shooting. It drives a two-axis (pan/tilt) camera rig with stepper motors, triggers a camera via GPIO, and computes a grid of shots based on lens parameters and desired overlap. Ideal for panoramas, photogrammetry, or timelapse sequences.

> **Work in progress** — This project is still under active development. Features may change, and some functionality may be incomplete or experimental.

## Hardware (Default Setup)

| Component | Description |
|-----------|-------------|
| **MCU** | Raspberry Pi Zero 2 WH |
| **Stepper drivers** | A4988 |
| **Stepper motors** | 2× NEMA-17 |
| **Voltage regulator** | Purecrea MT3608 (step-up/boost) |

The software supports a Nikon D90 (or compatible) camera via GPIO: focus and shutter lines are connected to the Pi for remote triggering.

## Requirements

- Go 1.25 or later
- Raspberry Pi OS (or compatible) when running on hardware
- Root or GPIO group access for GPIO (rpio)

## Building

```bash
# Clone the repository
git clone https://github.com/cjeanneret/PanGo.git
cd PanGo

# Build
go build -o pango ./cmd/pango
```

### Cross-compilation for Raspberry Pi

```bash
# 32-bit ARM (Raspberry Pi Zero 2 W, etc.)
GOOS=linux GOARCH=arm GOARM=7 go build -o pango ./cmd/pango

# 64-bit ARM (Raspberry Pi 4/5, etc.)
GOOS=linux GOARCH=arm64 go build -o pango ./cmd/pango
```

## Usage

### Config file

Edit `configs/default.yaml` to match your hardware (GPIO pins, lens, overlap, angles, etc.).

### Run a single capture

```bash
./pango -config configs/default.yaml
```

### Run with web interface

```bash
# Default port 8080
./pango -web

# Custom port
./pango -web 8980
```

Open `http://<raspberry-pi-ip>:8080` in a browser to control the rig and start a grid capture.

### CLI overrides

```bash
./pango -horizontal_angle_deg 180 -vertical_angle_deg 30 -focal_length_mm 35
```

### Mock GPIO (development without hardware)

In `configs/default.yaml`, set:

```yaml
defaults:
  mock_gpio: true
```

This allows testing on a PC without a Raspberry Pi.

## Dependencies and Licenses

| Package | License |
|---------|---------|
| [github.com/stianeikeland/go-rpio/v4](https://github.com/stianeikeland/go-rpio) | MIT |
| [gopkg.in/yaml.v3](https://gopkg.in/yaml.v3) | MIT / Apache 2.0 |

## AI-Assisted Development

This project was developed with AI assistance using **Cursor** (Composer agent).

## License

MIT License — see [LICENSE](LICENSE) for details.
