# PanGo Logo

The logo (`logo-pango.png`) was generated using **Leonardo.ai**.

## Generation details

| Field | Value |
|-------|-------|
| Generator | [Leonardo.ai](https://leonardo.ai) |
| Model | Lucid Origin |
| Style | Dynamic |

## Prompt

> Using the description of the PanGo project, create a technical yet slim logo for the project. It must avoid any "instagram-like" general shape and form, but must still be able to represent a "gigapixel head", to create gigapixel photography using a camera.
> The project description is: PanGo is a pan-tilt photography controller for automated grid shooting. It drives a two-axis (pan/tilt) camera rig with stepper motors, triggers a camera via GPIO, and computes a grid of shots based on lens parameters and desired overlap. Ideal for panoramas, photogrammetry, or timelapse sequences.
>
> You can read more on its github project page: https://github.com/cjeanneret/pango if you can access it.

## Post-processing

The raw output (692×900) includes the hexagon icon and "PanGo" text below it. The text was cropped out (top 710 px kept) and the result force-resized to 692×692 to produce a square source image. This introduces a negligible 2.5 % vertical squish.

```bash
magick logo.png -crop 692x710+0+0 +repage -resize 692x692! logo-pango.png
```

## Web variants

The following sizes are derived from the square source (692×692) using ImageMagick:

| File | Size | Use |
|------|------|-----|
| `favicon.ico` | 16, 32, 48 | Browser favicon |
| `favicon-16x16.png`, `favicon-32x32.png`, `favicon-48x48.png` | 16×16, 32×32, 48×48 | Favicon |
| `apple-touch-icon.png` | 180×180 | iOS / Android home screen (flattened on white) |
| `logo-64.png` … `logo-512.png` | 64 to 512 px | Web interface, PWA |
