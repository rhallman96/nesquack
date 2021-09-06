package system

// mirrorMode describes how memory is mirrored in PPU name tables
// according to a 2x2 grid configuration.
type mirrorMode int

// mirroring modes
const (
	onePage = iota
	horizontal
	vertical
	onePageHigh
)

// index returns the array index for mirrored name table accesses.
func (m mirrorMode) index(a, start, mirror uint16) uint16 {
	i := (a - start) % nameTableSectionSize
	switch m {
	case onePage:
		return i % mirror
	case onePageHigh:
		return mirror + (i % mirror)
	case horizontal:
		if i < (2 * mirror) {
			return i % mirror
		}
		return (i % mirror) + mirror
	case vertical:
		return i % (2 * mirror)
	}
	return 0
}

// mirrorIndex returns an array index for simple mirrored memory accesses.
// This is equivalent to single page mirroring in the PPU.
func mirrorIndex(a, start, mirror uint16) uint16 {
	return (a - start) % mirror
}
