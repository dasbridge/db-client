package hsb

import "math"

func HsbToRGB(hu, sa, br float64) (r, g, b uint8) {
	hu /= 60

	sa = sa * (255 / 100)
	br = br * (255 / 100)

	maxRGB := br

	if 0 == sa {
		r = 0
		g = 0
		b = 0

		return
	}

	delta := sa * maxRGB / 255

	if hu > 3 {
		b = uint8(maxRGB + 0.5)

		if hu > 3 {
			g = uint8(maxRGB - delta + 0.5)
			r = uint8((hu - 4) * delta + 0.5) + g
		} else {
			r = uint8(maxRGB - delta + 0.5)
			g = uint8(float64(r) - math.Trunc((hu - 4) * delta + 0.5))
		}
	} else {
	}



	return
}
