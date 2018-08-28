package s2tools

import (
	"sort"

	"github.com/golang/geo/s2"
)

// CellUnionUnion Union of CellUnion
// Note that this is note the same as a normalized CellUnion
// even if some cells can be merged by parent we return all the cells as the original level
func CellUnionUnion(src s2.CellUnion, others ...s2.CellUnion) s2.CellUnion {
	m := make(map[s2.CellID]struct{})
	for _, c := range src {
		m[c] = struct{}{}
	}

	for _, cu := range others {
		for _, c := range cu {
			m[c] = struct{}{}
		}
	}
	var res s2.CellUnion
	for k := range m {
		res = append(res, k)
	}
	sort.Slice(res, func(i, j int) bool { return uint64(res[i]) < uint64(res[j]) })
	return res
}

// CellUnionMissing returns cell from `from` not present in `to`
func CellUnionMissing(from, to s2.CellUnion) s2.CellUnion {
	m := make(map[s2.CellID]int)

	for _, c := range to {
		m[c]++
	}

	var ret []s2.CellID
	for _, c := range from {
		if m[c] > 0 {
			m[c]--
			continue
		}
		ret = append(ret, c)
	}

	return ret
}
