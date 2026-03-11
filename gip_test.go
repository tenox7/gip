package gip

import (
	"bytes"
	"fmt"
	"image"
	"image/color/palette"
	"image/gif"
	"image/png"
	"os"
	"testing"
	"time"
)

func TestSpeedComparison(t *testing.T) {
	// Load test image
	pngFile, err := os.Open("test.png")
	if err != nil {
		t.Fatalf("Failed to open test.png: %v", err)
	}
	defer pngFile.Close()
	
	img, err := png.Decode(pngFile)
	if err != nil {
		t.Fatalf("Failed to decode PNG: %v", err)
	}
	
	// Test standard library
	var stdBuf bytes.Buffer
	stdStart := time.Now()
	err = gif.Encode(&stdBuf, img, nil)
	stdTime := time.Since(stdStart)
	if err != nil {
		t.Fatalf("Standard library encode failed: %v", err)
	}
	
	// Test our fast encoder
	var fastBuf bytes.Buffer
	fastStart := time.Now()
	err = Encode(&fastBuf, img, nil)
	fastTime := time.Since(fastStart)
	if err != nil {
		t.Fatalf("Fast encode failed: %v", err)
	}
	
	// Save both GIFs for visual comparison
	if err := os.WriteFile("test_standard.gif", stdBuf.Bytes(), 0644); err != nil {
		t.Errorf("Failed to write standard GIF: %v", err)
	}
	
	if err := os.WriteFile("test_fast.gif", fastBuf.Bytes(), 0644); err != nil {
		t.Errorf("Failed to write fast GIF: %v", err)
	}
	
	// Print results
	fmt.Printf("\n=== GIF Encoding Speed Comparison ===\n")
	fmt.Printf("Standard library: %v\n", stdTime)
	fmt.Printf("Fast encoder:     %v\n", fastTime)
	fmt.Printf("Speedup:          %.2fx\n", float64(stdTime)/float64(fastTime))
	fmt.Printf("\nFile sizes:\n")
	fmt.Printf("Standard: %d bytes (test_standard.gif)\n", stdBuf.Len())
	fmt.Printf("Fast:     %d bytes (test_fast.gif) - %.1f%%\n", fastBuf.Len(), float64(fastBuf.Len())/float64(stdBuf.Len())*100)
	fmt.Printf("\nFiles saved for visual comparison:\n")
	fmt.Printf("- test_standard.gif (standard library)\n")
	fmt.Printf("- test_fast.gif (our fast encoder)\n")
}

// 3-Way Performance Comparison Test
func TestThreeWaySpeedComparison(t *testing.T) {
	// Load test image
	pngFile, err := os.Open("test.png")
	if err != nil {
		t.Fatalf("Failed to open test.png: %v", err)
	}
	defer pngFile.Close()
	
	img, err := png.Decode(pngFile)
	if err != nil {
		t.Fatalf("Failed to decode PNG: %v", err)
	}
	
	// Test 1: Standard GIF library (baseline)
	var stdBuf bytes.Buffer
	stdStart := time.Now()
	err = gif.Encode(&stdBuf, img, nil)
	stdTime := time.Since(stdStart)
	if err != nil {
		t.Fatalf("Standard library encode failed: %v", err)
	}
	
	// Test 2: Standard GIF library with 216-color fast LUT (no parallelization)
	var fastLutBuf bytes.Buffer
	fastLutStart := time.Now()
	paletted216 := convert216ColorLUT(img)
	err = gif.Encode(&fastLutBuf, paletted216, nil)
	fastLutTime := time.Since(fastLutStart)
	if err != nil {
		t.Fatalf("Fast LUT encode failed: %v", err)
	}
	
	// Test 3: Our parallel GIP encoder
	var gipBuf bytes.Buffer
	gipStart := time.Now()
	err = Encode(&gipBuf, img, nil)
	gipTime := time.Since(gipStart)
	if err != nil {
		t.Fatalf("GIP parallel encode failed: %v", err)
	}
	
	// Save all three GIFs for visual comparison
	if err := os.WriteFile("test_standard.gif", stdBuf.Bytes(), 0644); err != nil {
		t.Errorf("Failed to write standard GIF: %v", err)
	}
	
	if err := os.WriteFile("test_216lut.gif", fastLutBuf.Bytes(), 0644); err != nil {
		t.Errorf("Failed to write 216-LUT GIF: %v", err)
	}
	
	if err := os.WriteFile("test_gip_parallel.gif", gipBuf.Bytes(), 0644); err != nil {
		t.Errorf("Failed to write GIP parallel GIF: %v", err)
	}
	
	// Print comprehensive results
	fmt.Printf("\n=== 3-Way GIF Encoding Performance Comparison ===\n")
	fmt.Printf("1. Standard library:        %v\n", stdTime)
	fmt.Printf("2. Standard + 216-LUT:      %v\n", fastLutTime)
	fmt.Printf("3. GIP parallel encoder:    %v\n", gipTime)
	fmt.Printf("\nSpeedup Analysis:\n")
	fmt.Printf("216-LUT vs Standard:        %.2fx faster\n", float64(stdTime)/float64(fastLutTime))
	fmt.Printf("GIP vs Standard:            %.2fx faster\n", float64(stdTime)/float64(gipTime))
	fmt.Printf("GIP vs 216-LUT:             %.2fx faster\n", float64(fastLutTime)/float64(gipTime))
	fmt.Printf("\nFile sizes:\n")
	fmt.Printf("Standard:     %d bytes (test_standard.gif)\n", stdBuf.Len())
	fmt.Printf("216-LUT:      %d bytes (test_216lut.gif) - %.1f%% of standard\n", fastLutBuf.Len(), float64(fastLutBuf.Len())/float64(stdBuf.Len())*100)
	fmt.Printf("GIP parallel: %d bytes (test_gip_parallel.gif) - %.1f%% of standard\n", gipBuf.Len(), float64(gipBuf.Len())/float64(stdBuf.Len())*100)
	fmt.Printf("\nFiles saved for visual comparison:\n")
	fmt.Printf("- test_standard.gif (standard library)\n")
	fmt.Printf("- test_216lut.gif (standard + 216-color LUT)\n")
	fmt.Printf("- test_gip_parallel.gif (GIP parallel encoder)\n")
}

// convert216ColorLUT converts an image to 216-color paletted image using fast LUT
// This mimics the implementation from ../wrp/util.go but without parallelization
func convert216ColorLUT(img image.Image) *image.Paletted {
	// FastGifLut maps 8-bit color values to 6 levels (0-5) for 216 web-safe colors
	var FastGifLut = [256]int{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5}
	
	r := img.Bounds()
	p := image.NewPaletted(r, palette.WebSafe)
	
	// Convert without parallelization (sequential processing)
	if i64, ok := img.(image.RGBA64Image); ok {
		for y := r.Min.Y; y < r.Max.Y; y++ {
			for x := r.Min.X; x < r.Max.X; x++ {
				c := i64.RGBA64At(x, y)
				r6 := FastGifLut[c.R>>8]
				g6 := FastGifLut[c.G>>8]
				b6 := FastGifLut[c.B>>8]
				p.SetColorIndex(x, y, uint8(36*r6+6*g6+b6))
			}
		}
	} else {
		for y := r.Min.Y; y < r.Max.Y; y++ {
			for x := r.Min.X; x < r.Max.X; x++ {
				c := img.At(x, y)
				cr, cg, cb, _ := c.RGBA()
				// RGBA() returns 16-bit values, we need 8-bit
				r6 := FastGifLut[(cr>>8)&0xff]
				g6 := FastGifLut[(cg>>8)&0xff]
				b6 := FastGifLut[(cb>>8)&0xff]
				p.SetColorIndex(x, y, uint8(36*r6+6*g6+b6))
			}
		}
	}
	
	return p
}

func BenchmarkStandardLibrary(b *testing.B) {
	pngFile, err := os.Open("test.png")
	if err != nil {
		b.Skip("test.png not found")
	}
	defer pngFile.Close()
	
	img, err := png.Decode(pngFile)
	if err != nil {
		b.Fatal(err)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		gif.Encode(&buf, img, nil)
	}
}

func Benchmark216ColorLUT(b *testing.B) {
	pngFile, err := os.Open("test.png")
	if err != nil {
		b.Skip("test.png not found")
	}
	defer pngFile.Close()
	
	img, err := png.Decode(pngFile)
	if err != nil {
		b.Fatal(err)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		paletted := convert216ColorLUT(img)
		gif.Encode(&buf, paletted, nil)
	}
}

func BenchmarkGIPParallel(b *testing.B) {
	pngFile, err := os.Open("test.png")
	if err != nil {
		b.Skip("test.png not found")
	}
	defer pngFile.Close()
	
	img, err := png.Decode(pngFile)
	if err != nil {
		b.Fatal(err)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		Encode(&buf, img, nil)
	}
}