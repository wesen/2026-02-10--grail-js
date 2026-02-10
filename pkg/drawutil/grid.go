package drawutil

import "github.com/wesen/grail/pkg/cellbuf"

// DrawGrid fills the buffer with grid dots ('·') at regular intervals,
// offset by camera position. Points where (worldX % spacingX == 0) and
// (worldY % spacingY == 0) get a dot.
func DrawGrid(buf *cellbuf.Buffer, camX, camY, spacingX, spacingY int, style cellbuf.StyleKey) {
	for r := 0; r < buf.H; r++ {
		wy := r + camY
		if mod(wy, spacingY) != 0 {
			continue
		}
		for c := 0; c < buf.W; c++ {
			wx := c + camX
			if mod(wx, spacingX) == 0 {
				buf.Set(c, r, '·', style)
			}
		}
	}
}

// mod returns a non-negative modulus (Go's % can return negative for negative operands).
func mod(a, m int) int {
	if m == 0 {
		return 0
	}
	r := a % m
	if r < 0 {
		r += m
	}
	return r
}
