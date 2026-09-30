package analysis

import (
	"sort"

	"github.com/RaviBharathi410/distributedtrace/internal/domain"
)

type SpanNode struct {
	Span       domain.Span
	Children   []*SpanNode
	Parent     *SpanNode
	SelfTimeMs float64
}

type CausalResult struct {
	RootCauseService string
	RootCausePath    []string
	RootCauseSpanID  string
	DeepestSelfTime  float64
}

type CausalAnalyzer struct {
	// Minimum percentage of parent duration a child must account for to attribute cause downstream
	ChildContributionThreshold float64
}

func NewCausalAnalyzer() *CausalAnalyzer {
	return &CausalAnalyzer{
		ChildContributionThreshold: 0.40, // 40% of parent duration
	}
}

// BuildTraceTree constructs a hierarchical tree from a flat slice of spans in a trace.
func (c *CausalAnalyzer) BuildTraceTree(spans []domain.Span) (*SpanNode, map[string]*SpanNode) {
	if len(spans) == 0 {
		return nil, nil
	}

	nodes := make(map[string]*SpanNode, len(spans))
	for _, s := range spans {
		nodes[s.SpanID] = &SpanNode{
			Span:     s,
			Children: []*SpanNode{},
		}
	}

	var root *SpanNode

	for _, node := range nodes {
		parentID := node.Span.ParentSpanID
		if parentID != "" && nodes[parentID] != nil {
			parent := nodes[parentID]
			node.Parent = parent
			parent.Children = append(parent.Children, node)
		} else {
			// If no parent or parent not found in span set, mark as root (or earliest start time)
			if root == nil || node.Span.StartTime.Before(root.Span.StartTime) {
				root = node
			}
		}
	}

	// Calculate SelfTime for all nodes
	for _, node := range nodes {
		var childrenDuration float64
		for _, child := range node.Children {
			childrenDuration += float64(child.Span.DurationMs)
		}
		selfTime := float64(node.Span.DurationMs) - childrenDuration
		if selfTime < 0 {
			selfTime = 0
		}
		node.SelfTimeMs = selfTime
	}

	return root, nodes
}

// AnalyzeRootCause traverses the trace tree to isolate the root-cause downstream service
// responsible for latency inflation.
func (c *CausalAnalyzer) AnalyzeRootCause(spans []domain.Span) CausalResult {
	if len(spans) == 0 {
		return CausalResult{}
	}

	root, _ := c.BuildTraceTree(spans)
	if root == nil {
		return CausalResult{
			RootCauseService: spans[0].ServiceName,
			RootCausePath:    []string{spans[0].ServiceName},
			RootCauseSpanID:  spans[0].SpanID,
			DeepestSelfTime:  float64(spans[0].DurationMs),
		}
	}

	curr := root
	path := []string{curr.Span.ServiceName}

	for {
		if len(curr.Children) == 0 {
			// Leaf node reached — the latency terminates here
			break
		}

		// Sort children by duration descending to find the critical path
		sort.Slice(curr.Children, func(i, j int) bool {
			return curr.Children[i].Span.DurationMs > curr.Children[j].Span.DurationMs
		})

		dominantChild := curr.Children[0]
		childDuration := float64(dominantChild.Span.DurationMs)
		parentDuration := float64(curr.Span.DurationMs)

		// If the parent's self-time is greater than the dominant child's duration,
		// the bottleneck is in the parent itself, not downstream.
		if curr.SelfTimeMs >= childDuration {
			break
		}

		// If dominant child represents a significant portion of parent duration, step down
		if parentDuration > 0 && (childDuration/parentDuration) >= c.ChildContributionThreshold {
			curr = dominantChild
			// Avoid duplicate adjacent service names in path (e.g. internal service RPCs)
			if len(path) == 0 || path[len(path)-1] != curr.Span.ServiceName {
				path = append(path, curr.Span.ServiceName)
			}
		} else {
			// Neither child dominates enough to shift blame downstream
			break
		}
	}

	return CausalResult{
		RootCauseService: curr.Span.ServiceName,
		RootCausePath:    path,
		RootCauseSpanID:  curr.Span.SpanID,
		DeepestSelfTime:  curr.SelfTimeMs,
	}
}
