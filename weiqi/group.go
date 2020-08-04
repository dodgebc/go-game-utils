package weiqi

// group tracks a group of connected stones
// it can be used immediately and reused easily
type group struct {
	edge     []vertex
	interior []vertex
	alive    bool
}

// expandAll finds all connected stones
func (p *group) expandAll(v vertex, b board) {
	p.edge = p.edge[:0]
	p.interior = p.interior[:0]
	p.alive = false
	p.edge = append(p.edge, v)

	for p.expand(b) > 0 {
	}
}

// expandAllIfDead finds all connected stones, unless the group is alive
func (p *group) expandAllIfDead(v vertex, b board) {
	p.edge = p.edge[:0]
	p.interior = p.interior[:0]
	p.alive = false
	p.edge = append(p.edge, v)

	for p.expand(b) > 0 && !p.alive {
	}
	if p.alive {
		p.edge = p.edge[:0]
		p.interior = p.interior[:0]
	}
}

// expand grows the group to include more connected stones and returns the number added
// when it returns 0, the group is complete
func (p *group) expand(b board) int {

	oldEdgeLen := len(p.edge)
	p.interior = append(p.interior, p.edge...) // Move edge to interior
	p.edge = p.edge[:0]                        // Reset edge

	// Loop over old edge of group
	for _, v := range p.interior[len(p.interior)-oldEdgeLen:] {
		vColor := b.flatArray[v[0]*b.width+v[1]]

		// Loop over adjacent vertices
		for i := 0; i < 2; i++ {
			for j := -1; j < 2; j += 2 {
				adj := vertex{v[0] + i*j, v[1] + (1-i)*j}
				if (adj[0] >= 0) && (adj[0] < b.height) && (adj[1] >= 0) && (adj[1] < b.width) {
					adjColor := b.flatArray[adj[0]*b.width+adj[1]]
					switch adjColor {
					case vColor: // Same color, maybe we should expand group to include
						shouldAdd := true
						for _, already := range p.interior { // Check if already in group
							if adj == already {
								shouldAdd = false
							}
						}
						for _, already := range p.edge { // Check if already added to new edge
							if adj == already {
								shouldAdd = false
							}
						}
						if shouldAdd {
							p.edge = append(p.edge, adj)
						}
					case 0: // Liberty, group is alive (SPEED: should be possible to return immediately if called by expandAllIfDead, probably insignificant)
						p.alive = true
					}
				}
			}
		}
	}

	return len(p.edge)
}
