package terraform

import (
	"fmt"

	"github.com/hashicorp/terraform/config/module"
	"github.com/hashicorp/terraform/dag"
)

// OrphanTransformer is a GraphTransformer that adds orphans to the
// graph. This transformer adds both resource and module orphans.
type OrphanTransformer struct {
	// State is the global state. We require the global state to
	// properly find module orphans at our path.
	State *State

	// Module is the root module. We'll look up the proper configuration
	// using the graph path.
	Module *module.Tree
}

func (t *OrphanTransformer) Transform(g *Graph) error {
	state := t.State.ModuleByPath(g.Path)
	if state == nil {
		// If there is no state for our module, there can't be any orphans
		return nil
	}

	module := t.Module.Child(g.Path[1:])
	if module == nil {
		panic(fmt.Sprintf(
			"module not found for path: %#v",
			g.Path[1:]))
	}
	config := module.Config()

	// Go over each resource orphan and add it to the graph.
	resourceOrphans := state.Orphans(config)
	resourceVertexes := make([]dag.Vertex, len(resourceOrphans))
	for i, k := range resourceOrphans {
		resourceVertexes[i] = g.Add(&graphNodeOrphanResource{
			ResourceName: k,
			dependentOn:  state.Resources[k].Dependencies,
		})
	}

	// Go over each module orphan and add it to the graph. We store the
	// vertexes and states outside so that we can connect dependencies later.
	moduleOrphans := t.State.ModuleOrphans(g.Path, config)
	moduleVertexes := make([]dag.Vertex, len(moduleOrphans))
	for i, path := range moduleOrphans {
		moduleVertexes[i] = g.Add(&graphNodeOrphanModule{
			Path:        path,
			dependentOn: t.State.ModuleByPath(path).Dependencies,
		})
	}

	// Now do the dependencies. We do this _after_ adding all the orphan
	// nodes above because there are cases in which the orphans themselves
	// depend on other orphans.

	// Resource dependencies
	for _, v := range resourceVertexes {
		g.ConnectDependent(v)
	}

	// Module dependencies
	for _, v := range moduleVertexes {
		g.ConnectDependent(v)
	}

	return nil
}

// graphNodeOrphanModule is the graph vertex representing an orphan resource..
type graphNodeOrphanModule struct {
	Path []string

	dependentOn []string
}

func (n *graphNodeOrphanModule) DependableName() []string {
	return []string{n.dependableName()}
}

func (n *graphNodeOrphanModule) DependentOn() []string {
	return n.dependentOn
}

func (n *graphNodeOrphanModule) Name() string {
	return fmt.Sprintf("%s (orphan)", n.dependableName())
}

func (n *graphNodeOrphanModule) dependableName() string {
	return fmt.Sprintf("module.%s", n.Path[len(n.Path)-1])
}

// graphNodeOrphanResource is the graph vertex representing an orphan resource..
type graphNodeOrphanResource struct {
	ResourceName string

	dependentOn []string
}

func (n *graphNodeOrphanResource) DependableName() []string {
	return []string{n.dependableName()}
}

func (n *graphNodeOrphanResource) DependentOn() []string {
	return n.dependentOn
}

func (n *graphNodeOrphanResource) Name() string {
	return fmt.Sprintf("%s (orphan)", n.ResourceName)
}

func (n *graphNodeOrphanResource) ProvidedBy() string {
	return resourceProvider(n.ResourceName)
}

// GraphNodeEvalable impl.
func (n *graphNodeOrphanResource) EvalTree() EvalNode {
	return nil
	/*
		TODO
			return &EvalSequence{
				Nodes: []EvalNode{
					&EvalRefresh{},
					&EvalDiff{},
					&EvalApply{},
					&EvalCommitState{},
				},
			}
	*/
}

func (n *graphNodeOrphanResource) dependableName() string {
	return n.ResourceName
}