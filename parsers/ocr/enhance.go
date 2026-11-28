package ocr

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"strings"
	"unicode"

	clipLayout "github.com/upendra7470/clip/internal/layout"
)

type imageVariant struct {
	path string
	name string
}

func prepareEnhancedVariants(path string) ([]imageVariant, func(), error) {
	primary, primaryCleanup, err := prepareImage(path, true)
	if err != nil {
		return nil, func() {}, err
	}
	cleanups := []func(){primaryCleanup}
	cleanup := func() {
		for index := len(cleanups) - 1; index >= 0; index-- {
			cleanups[index]()
		}
	}

	// #nosec G304 -- this is the private, validated image prepared above.
	file, err := os.Open(primary)
	if err != nil {
		cleanup()
		return nil, func() {}, fmt.Errorf("open enhanced OCR image: %w", err)
	}
	decoded, _, err := image.Decode(file)
	_ = file.Close()
	if err != nil {
		cleanup()
		return nil, func() {}, fmt.Errorf("decode enhanced OCR image: %w", err)
	}
	gray := grayscale(decoded)
	thresholded := otsuBinarize(gray)
	thresholdPath, thresholdCleanup, err := writeVariant("threshold", thresholded)
	if err != nil {
		cleanup()
		return nil, func() {}, err
	}
	cleanups = append(cleanups, thresholdCleanup)
	variants := []imageVariant{
		{path: primary, name: "contrast"},
		{path: thresholdPath, name: "threshold"},
	}

	angle := estimateSkew(thresholded)
	if math.Abs(angle) >= 0.5 {
		deskewed := rotateSmall(thresholded, angle)
		deskewedPath, deskewCleanup, err := writeVariant("deskew", deskewed)
		if err != nil {
			cleanup()
			return nil, func() {}, err
		}
		cleanups = append(cleanups, deskewCleanup)
		variants = append(variants, imageVariant{path: deskewedPath, name: "deskew"})
	}
	return variants, cleanup, nil
}

func recognizeEnhanced(ctx context.Context, runner Runner, variants []imageVariant, mode clipLayout.Mode) (string, error) {
	if len(variants) == 0 {
		return "", fmt.Errorf("enhanced OCR has no prepared image variants")
	}
	bestText, bestPath := "", ""
	bestScore := -int(^uint(0)>>1) - 1
	var firstErr error
	for _, variant := range variants {
		text, err := recognizeWithLayout(ctx, runner, variant.path, mode)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			if ctx.Err() != nil {
				return "", ctx.Err()
			}
			continue
		}
		if score := textQualityScore(text); score > bestScore {
			bestText, bestPath, bestScore = text, variant.path, score
		}
	}
	if bestPath == "" {
		return "", firstErr
	}
	for _, degrees := range []int{90, 180, 270} {
		rotated, cleanup, err := rotatedImage(bestPath, degrees)
		if err != nil {
			return "", err
		}
		candidate, recognizeErr := recognizeWithLayout(ctx, runner, rotated, mode)
		cleanup()
		if recognizeErr != nil {
			if ctx.Err() != nil {
				return "", ctx.Err()
			}
			continue
		}
		if score := textQualityScore(candidate); score > bestScore {
			bestText, bestScore = candidate, score
		}
	}
	return bestText, nil
}

func textQualityScore(text string) int {
	score := 0
	letters := 0
	noise := 0
	for _, value := range text {
		switch {
		case unicode.IsLetter(value):
			letters++
			score += 3
		case unicode.IsDigit(value):
			score += 2
		case unicode.IsSpace(value):
			score++
		case unicode.IsPunct(value):
			// Normal punctuation helps slightly; long punctuation-only OCR
			// artifacts are penalized by the word and noise terms below.
			score++
		case unicode.IsControl(value) || value == unicode.ReplacementChar:
			noise++
		}
	}
	words := strings.Fields(text)
	for _, word := range words {
		wordLetters := 0
		for _, value := range word {
			if unicode.IsLetter(value) || unicode.IsDigit(value) {
				wordLetters++
			}
		}
		if wordLetters >= 2 {
			score += 5
		} else if wordLetters == 0 {
			score -= 4
		}
	}
	if letters == 0 {
		score -= 20
	}
	return score - noise*10
}

func grayscale(source image.Image) *image.Gray {
	bounds := source.Bounds()
	gray := image.NewGray(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	for y := 0; y < bounds.Dy(); y++ {
		for x := 0; x < bounds.Dx(); x++ {
			gray.Set(x, y, source.At(bounds.Min.X+x, bounds.Min.Y+y))
		}
	}
	return gray
}

func otsuBinarize(source *image.Gray) *image.Gray {
	var histogram [256]int
	var totalValue int64
	for _, value := range source.Pix {
		histogram[value]++
		totalValue += int64(value)
	}
	pixels := len(source.Pix)
	if pixels == 0 {
		return image.NewGray(source.Bounds())
	}
	var sumAll float64
	for value, count := range histogram {
		sumAll += float64(value * count)
	}
	var sumBackground float64
	background := 0
	bestVariance := -1.0
	threshold := 127
	for value, count := range histogram {
		background += count
		if background == 0 {
			continue
		}
		foreground := pixels - background
		if foreground == 0 {
			break
		}
		sumBackground += float64(value * count)
		backgroundMean := sumBackground / float64(background)
		foregroundMean := (sumAll - sumBackground) / float64(foreground)
		difference := backgroundMean - foregroundMean
		variance := float64(background*foreground) * difference * difference
		if variance > bestVariance {
			bestVariance, threshold = variance, value
		}
	}
	darkBackground := totalValue/int64(pixels) < 128
	result := image.NewGray(source.Bounds())
	for y := 0; y < source.Bounds().Dy(); y++ {
		for x := 0; x < source.Bounds().Dx(); x++ {
			value := source.GrayAt(x, y).Y
			ink := value <= uint8(threshold)
			if darkBackground {
				ink = !ink
			}
			if ink {
				result.SetGray(x, y, color.Gray{Y: 0})
			} else {
				result.SetGray(x, y, color.Gray{Y: 255})
			}
		}
	}
	return result
}

func estimateSkew(source *image.Gray) float64 {
	bounds := source.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	if width < 32 || height < 32 {
		return 0
	}
	step := max(width, height)/1200 + 1
	centerX := float64(width-1) / 2
	bestAngle, bestScore := 0.0, -1.0
	for angle := -3.0; angle <= 3.0; angle += 0.5 {
		projection := make([]int, height+width/8+4)
		offset := len(projection)/2 - height/2
		tangent := math.Tan(angle * math.Pi / 180)
		for y := 0; y < height; y += step {
			for x := 0; x < width; x += step {
				if source.GrayAt(x, y).Y > 64 {
					continue
				}
				row := int(math.Round(float64(y)+tangent*(float64(x)-centerX))) + offset
				if row >= 0 && row < len(projection) {
					projection[row]++
				}
			}
		}
		score := 0.0
		for _, count := range projection {
			score += float64(count * count)
		}
		if score > bestScore || (score == bestScore && math.Abs(angle) < math.Abs(bestAngle)) {
			bestAngle, bestScore = angle, score
		}
	}
	return bestAngle
}

func rotateSmall(source *image.Gray, degrees float64) *image.Gray {
	bounds := source.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	result := image.NewGray(image.Rect(0, 0, width, height))
	for index := range result.Pix {
		result.Pix[index] = 255
	}
	radians := degrees * math.Pi / 180
	sine, cosine := math.Sin(radians), math.Cos(radians)
	cx, cy := float64(width-1)/2, float64(height-1)/2
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			dx, dy := float64(x)-cx, float64(y)-cy
			sourceX := cosine*dx + sine*dy + cx
			sourceY := -sine*dx + cosine*dy + cy
			sx, sy := int(math.Round(sourceX)), int(math.Round(sourceY))
			if sx >= 0 && sx < width && sy >= 0 && sy < height {
				result.SetGray(x, y, source.GrayAt(sx, sy))
			}
		}
	}
	return result
}

func writeVariant(name string, value image.Image) (string, func(), error) {
	file, err := os.CreateTemp("", "clip-ocr-"+name+"-*.png")
	if err != nil {
		return "", func() {}, fmt.Errorf("create %s OCR variant: %w", name, err)
	}
	path := file.Name()
	cleanup := func() { _ = os.Remove(path) }
	if err := png.Encode(file, value); err != nil {
		_ = file.Close()
		cleanup()
		return "", func() {}, fmt.Errorf("encode %s OCR variant: %w", name, err)
	}
	if err := file.Close(); err != nil {
		cleanup()
		return "", func() {}, fmt.Errorf("close %s OCR variant: %w", name, err)
	}
	return path, cleanup, nil
}
