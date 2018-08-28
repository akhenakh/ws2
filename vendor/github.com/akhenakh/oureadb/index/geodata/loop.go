package geodata

import "github.com/golang/geo/s2"

// LoopFromCoordinates creates a LoopFence from a list of lng lat
func LoopFromCoordinates(c []float64) *s2.Loop {
	if len(c)%2 != 0 || len(c) < 2*3 {
		return nil
	}
	points := make([]s2.Point, len(c)/2)

	for i := 0; i < len(c); i += 2 {
		points[i/2] = s2.PointFromLatLng(s2.LatLngFromDegrees(c[i+1], c[i]))
	}

	if points[0] == points[len(points)-1] {
		// remove last item if same as 1st
		points = append(points[:len(points)-1], points[len(points)-1+1:]...)
	}

	if s2.RobustSign(points[0], points[1], points[2]) != s2.CounterClockwise {
		// reversing the slice
		for i := len(points)/2 - 1; i >= 0; i-- {
			opp := len(points) - 1 - i
			points[i], points[opp] = points[opp], points[i]
		}
	}

	loop := s2.LoopFromPoints(points)
	return loop
}
