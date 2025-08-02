package gifp

import (
	"bytes"
	"fmt"
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

func BenchmarkFastEncoder(b *testing.B) {
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