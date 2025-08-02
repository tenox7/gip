# gifp - Fast Parallel GIF Encoder

A blazing fast GIF encoder that achieves **70x faster** encoding than Go's standard library by using 216 web-safe colors and true parallel processing.

## Features

- **70x Faster**: Eliminates dithering for true parallel color mapping
- **Smaller Files**: Produces files ~30% the size of standard encoding
- **True Parallelization**: Utilizes all CPU cores for color mapping
- **Simple API**: Drop-in replacement for basic GIF encoding needs

## Installation

```bash
go get github.com/tenox7/gifp
```

## Usage

```go
import "github.com/tenox7/gifp"

// Basic usage
err := gifp.Encode(w, img, nil)

// With custom worker count
err := gifp.Encode(w, img, &gifp.Options{
    Workers: 8,
})
```

## Benchmarks

```bash
go test -v
```

## How It Works

- Uses a fixed 216-color web-safe palette (6x6x6 color cube)
- Eliminates Floyd-Steinberg dithering for true parallel processing
- Each worker processes different image rows independently
- Fast color mapping using lookup table

## Trade-offs

This encoder prioritizes speed over color accuracy. It's perfect for:
- Web graphics where speed matters
- Thumbnails and previews
- Real-time GIF generation
- Server-side image processing

Not recommended for:
- Photographic images requiring high color fidelity
- Images that need dithering for smooth gradients

## Credits

The fast color quantization technique by Hill Ma https://github.com/mahiuchun

## License

Same as Go's standard library (BSD-style).