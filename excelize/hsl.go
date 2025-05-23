package excelize

import (
	"image/color"
	"math"
)

// HSLModel converts any color.Color to a HSL color.
var HSLModel = color.ModelFunc(hslModel)

// HSL represents a cylindrical coordinate of points in an RGB color model.
//
// Values are in the range 0 to 1.
type HSL struct {
	H, S, L float64
}

// RGBA returns the alpha-premultiplied red, green, blue and alpha values
// for the HSL.
func (c HSL) RGBA() (uint32, uint32, uint32, uint32) {
	r, g, b := HSLToRGB(c.H, c.S, c.L)
	return uint32(r) * 0x101, uint32(g) * 0x101, uint32(b) * 0x101, 0xffff
}

// hslModel converts a color.Color to HSL.
func hslModel(c color.Color) color.Color {
	if _, ok := c.(HSL); ok {
		return c
	}
	r, g, b, _ := c.RGBA()
	h, s, l := RGBToHSL(uint8(r>>8), uint8(g>>8), uint8(b>>8))
	return HSL{h, s, l}
}

// RGBToHSL converts an RGB triple to an HSL triple.
func RGBToHSL(r, g, b uint8) (h, s, l float64) {
	fR := float64(r) / 255
	fG := float64(g) / 255
	fB := float64(b) / 255
	maxVal := math.Max(math.Max(fR, fG), fB)
	minVal := math.Min(math.Min(fR, fG), fB)
	l = (maxVal + minVal) / 2
	if maxVal == minVal {
		// Achromatic.
		h, s = 0, 0
	} else {
		// Chromatic.
		d := maxVal - minVal
		if l > 0.5 {
			s = d / (2.0 - maxVal - minVal)
		} else {
			s = d / (maxVal + minVal)
		}
		switch maxVal {
		case fR:
			h = (fG - fB) / d
			if fG < fB {
				h += 6
			}
		case fG:
			h = (fB-fR)/d + 2
		case fB:
			h = (fR-fG)/d + 4
		}
		h /= 6
	}
	return
}

// HSLToRGB converts an HSL triple to an RGB triple.
func HSLToRGB(h, s, l float64) (r, g, b uint8) {
	var fR, fG, fB float64
	if s == 0 {
		fR, fG, fB = l, l, l
	} else {
		var q float64
		if l < 0.5 {
			q = l * (1 + s)
		} else {
			q = l + s - s*l
		}
		p := 2*l - q
		fR = hueToRGB(p, q, h+1.0/3)
		fG = hueToRGB(p, q, h)
		fB = hueToRGB(p, q, h-1.0/3)
	}
	r = uint8((fR * 255) + 0.5)
	g = uint8((fG * 255) + 0.5)
	b = uint8((fB * 255) + 0.5)
	return
}

// hueToRGB is a helper function for HSLToRGB.
func hueToRGB(p, q, t float64) float64 {
	if t < 0 {
		t++
	}
	if t > 1 {
		t--
	}
	if t < 1.0/6 {
		return p + (q-p)*6*t
	}
	if t < 0.5 {
		return q
	}
	if t < 2.0/3 {
		return p + (q-p)*(2.0/3-t)*6
	}
	return p
}
