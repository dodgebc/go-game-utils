package weiqi

// initial capacities before expansion
// (these don't matter too much)
const (
	edgeCap     = 2
	interiorCap = 2
)

// group tracks a group of connected stones
type group struct {
	edge     []vertex
	interior []vertex
	alive    bool
}

// newGroup finds all the connected stones from a vertex
func newGroup(v vertex, b board) group {
	var p group
	p.edge = make([]vertex, 1, edgeCap)
	p.edge[0] = v
	p.interior = make([]vertex, 0, interiorCap)
	for p.expand(b) > 0 {
		continue
	}
	return p
}

// newGroupIfDead stops expanding if group is confirmed alive (for speed)
func newGroupIfDead(v vertex, b board) group {
	var p group
	p.edge = make([]vertex, 1, edgeCap)
	p.edge[0] = v
	p.interior = make([]vertex, 0, interiorCap)
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

	oldEdgeLen := len(p.edge)
	p.interior = append(p.interior, p.edge...) // Move edge to interior
	p.edge = p.edge[:0]                        // Reset edge

	// Loop over old edge of group
	for _, v := range p.interior[len(p.interior)-oldEdgeLen:] {
		vColor := b.look(v)

		// Loop over adjacent vertices
		for i := 0; i < 2; i++ {
			for j := -1; j < 2; j += 2 {
				adj := vertex{v[0] + i*j, v[1] + (1-i)*j}
				if (adj[0] >= 0) && (adj[0] < b.height) && (adj[1] >= 0) && (adj[1] < b.width) {
					adjColor := b.look(adj) //b.flatArray[adj[0]*b.width+adj[1]]
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
					case 0: // Liberty, group is alive
						p.alive = true
					}
				}
			}
		}
	}

	return len(p.edge)
}
