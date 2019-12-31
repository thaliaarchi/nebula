package digraph // import "github.com/andrewarchi/nebula/digraph"

// Digraph is a directed graph.
type Digraph []graphNode

type graphNode struct {
	Edges   []int
	Visited bool
}

// AddEdge adds a directed edge from node i to j.
func (g Digraph) AddEdge(i, j int) {
	g[i].Edges = append(g[i].Edges, j)
}

// SCCs computes the strongly connected components of a graph.
func (g Digraph) SCCs() [][]int {
	postOrder := g.Reverse().PostOrder()
	var sccs [][]int
	for i := len(postOrder) - 1; i >= 0; i-- {
		if !g[postOrder[i]].Visited {
			sccs = append(sccs, g.visit(postOrder[i], nil))
		}
	}
	return sccs
}

// PostOrder traverses the graph with depth first search and returns the
// post-order traversal numbers.
func (g Digraph) PostOrder() []int {
	var postOrder []int
	for i := range g {
		postOrder = g.visit(i, postOrder)
	}
	return postOrder
}

func (g Digraph) visit(node int, postOrder []int) []int {
	if g[node].Visited {
		return postOrder
	}
	g[node].Visited = true
	for _, edge := range g[node].Edges {
		postOrder = g.visit(edge, postOrder)
	}
	return append(postOrder, node)
}

// Reverse creates the reverse graph of g.
func (g Digraph) Reverse() Digraph {
	r := make(Digraph, len(g))
	for node := range g {
		for _, edge := range g[node].Edges {
			r[edge].Edges = append(r[edge].Edges, node)
		}
	}
	return r
}

// ClearVisited resets the visited flags.
func (g Digraph) ClearVisited() {
	for i := range g {
		g[i].Visited = false
	}
}
