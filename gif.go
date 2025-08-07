package gip

import (
	"bufio"
	"bytes"
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

var FastGifLut = [256]int{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5}

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
	mapColors(pm, m, workers)
	
	if pm.Rect.Min != (image.Point{}) {
		dup := *pm
		dup.Rect = dup.Rect.Sub(dup.Rect.Min)
		pm = &dup
	}
	
	return encodeGIF(w, pm, workers)
}

func mapColors(pm *image.Paletted, m image.Image, workers int) {
	b := m.Bounds()
	height := b.Dy()
	rowsPerWorker := height / workers
	if rowsPerWorker < 1 {
		rowsPerWorker = 1
		workers = height
	}
	
	var wg sync.WaitGroup
	mapFunc := getColorMapper(m)
	
	for i := 0; i < workers; i++ {
		startY := b.Min.Y + i*rowsPerWorker
		endY := startY + rowsPerWorker
		if i == workers-1 {
			endY = b.Max.Y
		}
		
		wg.Add(1)
		go func(sy, ey int) {
			defer wg.Done()
			mapFunc(pm, m, b.Min.X, b.Max.X, sy, ey)
		}(startY, endY)
	}
	wg.Wait()
}

func getColorMapper(m image.Image) func(*image.Paletted, image.Image, int, int, int, int) {
	if i64, ok := m.(image.RGBA64Image); ok {
		return func(pm *image.Paletted, _ image.Image, minX, maxX, startY, endY int) {
			for y := startY; y < endY; y++ {
				for x := minX; x < maxX; x++ {
					c := i64.RGBA64At(x, y)
					idx := 36*FastGifLut[c.R>>8] + 6*FastGifLut[c.G>>8] + FastGifLut[c.B>>8]
					pm.SetColorIndex(x, y, uint8(idx))
				}
			}
		}
	}
	return func(pm *image.Paletted, m image.Image, minX, maxX, startY, endY int) {
		for y := startY; y < endY; y++ {
			for x := minX; x < maxX; x++ {
				r, g, b, _ := m.At(x, y).RGBA()
				idx := 36*FastGifLut[(r>>8)&0xff] + 6*FastGifLut[(g>>8)&0xff] + FastGifLut[(b>>8)&0xff]
				pm.SetColorIndex(x, y, uint8(idx))
			}
		}
	}
}

func encodeGIF(w io.Writer, pm *image.Paletted, workers int) error {
	if len(pm.Palette) == 0 {
		return errors.New("gif: cannot encode image block with empty palette")
	}

	bw := bufio.NewWriter(w)
	b := pm.Bounds()
	
	bw.WriteString("GIF89a")
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
	if n := len(pm.Palette); n > 0 {
		for litWidth = 2; litWidth < 8 && 1<<uint(litWidth) < n; litWidth++ {
		}
	}

	bw.WriteByte(uint8(litWidth))

	if err := writeLZWData(bw, pm, litWidth, workers); err != nil {
		return err
	}

	bw.WriteByte(0x00)
	bw.WriteByte(0x3B)
	return bw.Flush()
}

func writeLZWData(w io.Writer, pm *image.Paletted, litWidth int, workers int) error {
	b := pm.Bounds()
	dx, dy := b.Dx(), b.Dy()

	if workers > 1 && dy >= workers*4 {
		return writeParallelLZW(w, pm, litWidth, workers)
	}

	stripHeight := dy / workers
	if stripHeight < 1 {
		stripHeight = 1
		workers = dy
	}
	
	imageData := make([]byte, dx*dy)
	var wg sync.WaitGroup
	
	for i := 0; i < workers; i++ {
		startY := i * stripHeight
		endY := startY + stripHeight
		if i == workers-1 {
			endY = dy
		}
		
		wg.Add(1)
		go func(sy, ey int) {
			defer wg.Done()
			for y := sy; y < ey; y++ {
				copy(imageData[y*dx:(y+1)*dx], pm.Pix[y*pm.Stride:y*pm.Stride+dx])
			}
		}(startY, endY)
	}
	wg.Wait()

	bw := &blockWriter{w: w}
	lzww := lzw.NewWriter(bw, lzw.LSB, litWidth)
	
	if _, err := lzww.Write(imageData); err != nil {
		lzww.Close()
		return err
	}
	
	if err := lzww.Close(); err != nil {
		return err
	}
	
	return bw.close()
}

func writeParallelLZW(w io.Writer, pm *image.Paletted, litWidth int, workers int) error {
	dx, dy := pm.Bounds().Dx(), pm.Bounds().Dy()
	stripHeight := dy / workers
	if stripHeight < 1 {
		stripHeight = 1
		workers = dy
	}
	
	type chunk struct {
		idx  int
		data []byte
		err  error
	}
	
	results := make(chan chunk, workers)
	var wg sync.WaitGroup
	
	for i := 0; i < workers; i++ {
		startY, endY := i*stripHeight, (i+1)*stripHeight
		if i == workers-1 {
			endY = dy
		}
		
		wg.Add(1)
		go func(idx, sy, ey int) {
			defer wg.Done()
			
			stripSize := (ey - sy) * dx
			stripData := make([]byte, stripSize)
			offset := 0
			for y := sy; y < ey; y++ {
				copy(stripData[offset:offset+dx], pm.Pix[y*pm.Stride:y*pm.Stride+dx])
				offset += dx
			}
			
			var buf bytes.Buffer
			bw := &blockWriter{w: &buf}
			lzww := lzw.NewWriter(bw, lzw.LSB, litWidth)
			
			if _, err := lzww.Write(stripData); err != nil {
				results <- chunk{idx, nil, err}
				return
			}
			
			if err := lzww.Close(); err != nil {
				results <- chunk{idx, nil, err}
				return
			}
			
			if err := bw.close(); err != nil {
				results <- chunk{idx, nil, err}
				return
			}
			
			results <- chunk{idx, buf.Bytes(), nil}
		}(i, startY, endY)
	}
	
	go func() {
		wg.Wait()
		close(results)
	}()
	
	chunks := make([][]byte, workers)
	for result := range results {
		if result.err != nil {
			return result.err
		}
		chunks[result.idx] = result.data
	}
	
	for _, data := range chunks {
		if _, err := w.Write(data); err != nil {
			return err
		}
	}
	
	return nil
}

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

func writeUint16(w io.Writer, v uint16) error {
	var buf [2]byte
	binary.LittleEndian.PutUint16(buf[:], v)
	_, err := w.Write(buf[:])
	return err
}

func writeColorTable(w io.Writer, p color.Palette, paddedSize int) error {
	for i := 0; i < paddedSize; i++ {
		var rgb [3]byte
		if i < len(p) {
			c := color.NRGBAModel.Convert(p[i]).(color.NRGBA)
			rgb = [3]byte{c.R, c.G, c.B}
		}
		if _, err := w.Write(rgb[:]); err != nil {
			return err
		}
	}
	return nil
}

func log2(x int) int {
	for i, v := range [8]int{2, 4, 8, 16, 32, 64, 128, 256} {
		if x <= v {
			return i
		}
	}
	return -1
}