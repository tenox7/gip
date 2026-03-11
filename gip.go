package gip

import (
	"bufio"
	"compress/lzw"
	"encoding/binary"
	"errors"
	"image"
	"image/color"
	"image/color/palette"
	"io"
	"runtime"
	"sync"
)

var FastGifLut = [256]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5}

var rLut [256]byte
var gLut [256]byte
var bLut [256]byte

func init() {
	for i := 0; i < 256; i++ {
		rLut[i] = FastGifLut[i] * 36
		gLut[i] = FastGifLut[i] * 6
		bLut[i] = FastGifLut[i]
	}
}

type Options struct {
	Workers int
}

func Encode(w io.Writer, m image.Image, o *Options) error {
	workers := runtime.NumCPU()
	if o != nil && o.Workers > 0 {
		workers = o.Workers
	}

	b := m.Bounds()
	if b.Dx() >= 1<<16 || b.Dy() >= 1<<16 {
		return errors.New("gif: image is too large to encode")
	}

	pm := image.NewPaletted(b, palette.WebSafe)

	height := b.Dy()
	rowsPerWorker := height / workers
	if rowsPerWorker < 1 {
		rowsPerWorker = 1
		workers = height
	}

	var wg sync.WaitGroup

	switch src := m.(type) {
	case *image.NRGBA:
		for i := 0; i < workers; i++ {
			startY := b.Min.Y + i*rowsPerWorker
			endY := startY + rowsPerWorker
			if i == workers-1 {
				endY = b.Max.Y
			}
			wg.Add(1)
			go func(sy, ey int) {
				defer wg.Done()
				for y := sy; y < ey; y++ {
					srcOff := (y-src.Rect.Min.Y)*src.Stride + (b.Min.X-src.Rect.Min.X)*4
					dstOff := (y-b.Min.Y)*pm.Stride
					for x := b.Min.X; x < b.Max.X; x++ {
						pm.Pix[dstOff] = rLut[src.Pix[srcOff]] + gLut[src.Pix[srcOff+1]] + bLut[src.Pix[srcOff+2]]
						srcOff += 4
						dstOff++
					}
				}
			}(startY, endY)
		}
	case *image.RGBA:
		for i := 0; i < workers; i++ {
			startY := b.Min.Y + i*rowsPerWorker
			endY := startY + rowsPerWorker
			if i == workers-1 {
				endY = b.Max.Y
			}
			wg.Add(1)
			go func(sy, ey int) {
				defer wg.Done()
				for y := sy; y < ey; y++ {
					srcOff := (y-src.Rect.Min.Y)*src.Stride + (b.Min.X-src.Rect.Min.X)*4
					dstOff := (y-b.Min.Y)*pm.Stride
					for x := b.Min.X; x < b.Max.X; x++ {
						pm.Pix[dstOff] = rLut[src.Pix[srcOff]] + gLut[src.Pix[srcOff+1]] + bLut[src.Pix[srcOff+2]]
						srcOff += 4
						dstOff++
					}
				}
			}(startY, endY)
		}
	default:
		for i := 0; i < workers; i++ {
			startY := b.Min.Y + i*rowsPerWorker
			endY := startY + rowsPerWorker
			if i == workers-1 {
				endY = b.Max.Y
			}
			wg.Add(1)
			go func(sy, ey int) {
				defer wg.Done()
				for y := sy; y < ey; y++ {
					dstOff := (y - b.Min.Y) * pm.Stride
					for x := b.Min.X; x < b.Max.X; x++ {
						r, g, bl, _ := m.At(x, y).RGBA()
						pm.Pix[dstOff] = rLut[byte(r>>8)] + gLut[byte(g>>8)] + bLut[byte(bl>>8)]
						dstOff++
					}
				}
			}(startY, endY)
		}
	}

	wg.Wait()

	if pm.Rect.Min != (image.Point{}) {
		dup := *pm
		dup.Rect = dup.Rect.Sub(dup.Rect.Min)
		pm = &dup
	}

	return encodeGIF(w, pm)
}

func encodeGIF(w io.Writer, pm *image.Paletted) error {
	if len(pm.Palette) == 0 {
		return errors.New("gif: cannot encode image block with empty palette")
	}

	bw := bufio.NewWriterSize(w, 32*1024)

	bw.WriteString("GIF89a")

	b := pm.Bounds()
	writeUint16(bw, uint16(b.Dx()))
	writeUint16(bw, uint16(b.Dy()))

	paddedSize := 1
	for paddedSize < len(pm.Palette) && paddedSize < 256 {
		paddedSize <<= 1
	}

	bw.WriteByte(0x80 | uint8(log2(paddedSize)))
	bw.WriteByte(0x00)
	bw.WriteByte(0x00)

	writeColorTable(bw, pm.Palette, paddedSize)

	bw.WriteByte(0x2C)
	writeUint16(bw, 0)
	writeUint16(bw, 0)
	writeUint16(bw, uint16(b.Dx()))
	writeUint16(bw, uint16(b.Dy()))
	bw.WriteByte(0x00)

	litWidth := 8
	n := len(pm.Palette)
	if n > 0 {
		for litWidth = 2; litWidth < 8 && 1<<uint(litWidth) < n; litWidth++ {
		}
	}

	bw.WriteByte(uint8(litWidth))

	if err := writeLZWData(bw, pm, litWidth); err != nil {
		return err
	}

	bw.WriteByte(0x00)
	bw.WriteByte(0x3B)

	return bw.Flush()
}

func writeLZWData(w io.Writer, pm *image.Paletted, litWidth int) error {
	dx := pm.Bounds().Dx()
	dy := pm.Bounds().Dy()

	data := pm.Pix
	if pm.Stride != dx {
		data = make([]byte, dx*dy)
		for y := 0; y < dy; y++ {
			copy(data[y*dx:(y+1)*dx], pm.Pix[y*pm.Stride:y*pm.Stride+dx])
		}
	} else {
		data = pm.Pix[:dx*dy]
	}

	bw := &blockWriter{w: w}
	lzww := lzw.NewWriter(bw, lzw.LSB, litWidth)

	if _, err := lzww.Write(data); err != nil {
		lzww.Close()
		return err
	}

	if err := lzww.Close(); err != nil {
		return err
	}

	return bw.close()
}

// blockWriter implements the GIF block structure for LZW data
type blockWriter struct {
	w   io.Writer
	buf [256]byte
	n   int
}

func (b *blockWriter) Write(p []byte) (int, error) {
	total := 0
	for len(p) > 0 {
		n := copy(b.buf[b.n+1:256], p)
		b.n += n
		p = p[n:]
		total += n
		
		if b.n == 255 {
			b.buf[0] = 255
			if _, err := b.w.Write(b.buf[:256]); err != nil {
				return total, err
			}
			b.n = 0
		}
	}
	return total, nil
}

func (b *blockWriter) close() error {
	if b.n > 0 {
		b.buf[0] = uint8(b.n)
		_, err := b.w.Write(b.buf[:b.n+1])
		return err
	}
	return nil
}

// Helper functions
func writeUint16(w io.Writer, v uint16) error {
	var buf [2]byte
	binary.LittleEndian.PutUint16(buf[:], v)
	_, err := w.Write(buf[:])
	return err
}

func writeColorTable(w io.Writer, p color.Palette, paddedSize int) error {
	var buf [768]byte
	for i := 0; i < paddedSize && i < len(p); i++ {
		c := color.NRGBAModel.Convert(p[i]).(color.NRGBA)
		buf[i*3] = c.R
		buf[i*3+1] = c.G
		buf[i*3+2] = c.B
	}
	_, err := w.Write(buf[:paddedSize*3])
	return err
}

// log2 returns the log2 of the smallest power of 2 >= x
func log2(x int) int {
	lookup := [8]int{2, 4, 8, 16, 32, 64, 128, 256}
	for i, v := range lookup {
		if x <= v {
			return i
		}
	}
	return -1
}