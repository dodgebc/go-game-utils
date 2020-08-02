package weiqi

// group tracks a group of connected stones
type group struct {
	edge     []vertex
	interior []vertex
	alive    bool
}

// newGroup finds all the connected stones from a vertex
func newGroup(v vertex, b board) group {
	p := group{edge: []vertex{v}}
	p.interior = make([]vertex, 0, 8)
	for p.expand(b) > 0 {
		continue
	}
	return p
}

// newGroupIfDead stops expanding if group is confirmed alive (for speed)
func newGroupIfDead(v vertex, b board) group {
	p := group{edge: []vertex{v}}
	p.interior = make([]vertex, 0, 8)
	for p.expand(b) > 0 {
		if p.alive {
			p.edge = p.edge[:0]
			p.interior = p.interior[:0]
			break
		}
	}
	return p
}

func (p *group) expand(b board) int {

	nextEdge := make([]vertex, 0, len(p.edge)*4)

	for _, v := range p.edge { // Loop over edge of group
		vColor := b.look(v)

		// big cost calling adjacent, maybe we should just do it in this function
		// could manually write out each coordinate and check if it's on the board
		// we would have to repeat a lot of code, not sure if there is a concise way to loop over 4
		for _, adj := range v.adjacent(b.height, b.width) { // Loop over adjacent vertices
			adjColor := b.look(adj)

			switch adjColor {
			case vColor: // Same color, maybe we should expand group to include
				shouldAdd := true
				for _, already := range nextEdge { // Check if already in group
					if adj.Equals(already) {
						shouldAdd = false
					}
				}
				for _, already := range p.edge { // Clumsy loop repeats, but this is fastest
					if adj.Equals(already) {
						shouldAdd = false
					}
				}
				for _, already := range p.interior {
					if adj.Equals(already) {
						shouldAdd = false
					}
				}
				if shouldAdd {
					nextEdge = append(nextEdge, adj)
				}
			case 0: // Liberty, group is alive
				p.alive = true
			}
		}
	}
	p.interior = append(p.interior, p.edge...) // Move edge to interior
	p.edge = nextEdge                          // Move new edge to edge
	return len(nextEdge)
}
