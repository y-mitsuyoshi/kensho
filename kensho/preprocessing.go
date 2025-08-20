package kensho

import (
	"bytes"
	"image"
	"image/jpeg"
	"image/png"
	"math"

	"github.com/anthonynsimon/bild/adjust"
	"github.com/anthonynsimon/bild/effect"
	"github.com/anthonynsimon/bild/transform"
	"golang.org/x/image/webp"
)

// PreprocessImage applies a series of preprocessing steps to an image.
func PreprocessImage(imgData []byte, mimeType string) ([]byte, error) {
	var (
		img image.Image
		err error
	)

	// Decode the image
	switch mimeType {
	case "image/jpeg":
		img, err = jpeg.Decode(bytes.NewReader(imgData))
	case "image/png":
		img, err = png.Decode(bytes.NewReader(imgData))
	case "image/webp":
		img, err = webp.Decode(bytes.NewReader(imgData))
	default:
		// Fallback for other image types
		img, _, err = image.Decode(bytes.NewReader(imgData))
	}

	if err != nil {
		// If decoding fails, return original data, as it might not be an image
		return imgData, nil
	}

	// Apply preprocessing steps
	img = deskew(img)
	img = adjustContrast(img, 1.5)
	img = sharpen(img)
	img = removeNoise(img)

	// Encode the processed image back to its original format
	buf := new(bytes.Buffer)
	switch mimeType {
	case "image/jpeg":
		err = jpeg.Encode(buf, img, &jpeg.Options{Quality: 90})
	case "image/png":
		err = png.Encode(buf, img)
	default:
		// Default to PNG for other types, including WEBP as there's no standard encoder
		err = png.Encode(buf, img)
	}

	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// sharpen applies an unsharp mask to the image.
func sharpen(img image.Image) image.Image {
	// Radius affects the size of the edges to enhance. Sigma affects the amount of sharpening.
	return effect.UnsharpMask(img, 1.0, 1.2)
}

// adjustContrast increases the image contrast.
// A factor of 1.5 is a moderate increase.
func adjustContrast(img image.Image, factor float64) image.Image {
	return adjust.Contrast(img, factor)
}

// removeNoise applies a median filter to reduce salt-and-pepper noise.
// A radius of 1.0 is a good starting point.
func removeNoise(img image.Image) image.Image {
	return effect.Median(img, 1.0)
}

// deskew attempts to correct the skew of an image by finding the dominant rotation angle.
func deskew(img image.Image) image.Image {
	// First, convert the image to grayscale as edge detection works on luminance.
	gray := effect.Grayscale(img)

	// Find the best angle
	angle := findBestSkewAngle(gray)

	// If the angle is not significant, don't rotate
	if math.Abs(angle) < 0.1 {
		return img
	}

	// Rotate the original image to correct the skew.
	// Using a black background color for the new areas after rotation.
	return transform.Rotate(img, angle, &transform.RotationOptions{
		ResizeBounds: false,
		Pivot:        nil, // Pivot at the center
	})
}

// findBestSkewAngle calculates the optimal skew angle for an image.
func findBestSkewAngle(grayImg image.Image) float64 {
	edges := effect.Sobel(grayImg)

	maxAngle := 10.0 // Max angle to check in degrees
	angleStep := 0.2 // Step for angle search
	var bestAngle float64
	maxScore := -1.0

	for angle := -maxAngle; angle <= maxAngle; angle += angleStep {
		rotated := transform.Rotate(edges, angle, nil)
		score := calculateProjectionScore(rotated)

		if score > maxScore {
			maxScore = score
			bestAngle = angle
		}
	}

	return bestAngle
}

// calculateProjectionScore computes a score based on the horizontal projection of pixel intensities.
// A higher score indicates a more "peaky" projection, which is characteristic of well-aligned text.
func calculateProjectionScore(img image.Image) float64 {
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	projection := make([]float64, height)

	for y := 0; y < height; y++ {
		sum := 0.0
		for x := 0; x < width; x++ {
			// Get the luminance value of the pixel
			// We can use the red channel of the grayscaled sobel image as a proxy for intensity
			r, _, _, _ := img.At(x+bounds.Min.X, y+bounds.Min.Y).RGBA()
			sum += float64(r)
		}
		projection[y] = sum
	}

	// Calculate the variance of the projection profile.
	// Higher variance means sharper peaks, which is what we want.
	mean := 0.0
	for _, p := range projection {
		mean += p
	}
	mean /= float64(len(projection))

	variance := 0.0
	for _, p := range projection {
		variance += math.Pow(p-mean, 2)
	}
	variance /= float64(len(projection))

	return variance
}
