package s2tools

import (
	"strconv"
	"strings"

	"github.com/golang/geo/s2"
)

func ParseCellID(cs string) *s2.CellID {
	// 2/3210 notation
	if strings.Contains(cs, "/") {
		face, err := strconv.Atoi(string(cs[0]))
		if err != nil || face < 0 || face > 5 {
			return nil
		}
		c := s2.CellIDFromFace(face)
		for _, b := range cs[2:] {
			child, err := strconv.Atoi(string(b))
			if err != nil || child < 0 || child > 3 {
				return nil
			}

			c = c.Children()[child]
		}
		return &c
	}

	c := s2.CellIDFromToken(cs)
	if c != s2.CellID(0) {
		return &c
	}

	id, err := strconv.ParseUint(cs, 10, 64)
	if err != nil {
		return nil
	}
	c = s2.CellID(id)
	return &c
}
