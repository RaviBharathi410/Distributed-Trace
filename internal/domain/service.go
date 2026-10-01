package domain

type ServiceNode struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	P99Ms           float64 `json:"p99"`
	Health          string  `json:"health"` // "healthy", "degraded", "critical"
	ActiveAnomalies int     `json:"activeAnomalies,omitempty"`
}

type ServiceEdge struct {
	Source       string  `json:"source"`
	Target       string  `json:"target"`
	RPS          float64 `json:"rps"`
	ErrorRatePct float64 `json:"errorRate"`
	P95Ms        float64 `json:"p95Ms,omitempty"`
	CriticalPath bool    `json:"criticalPath,omitempty"`
}

type ServiceGraph struct {
	Nodes []ServiceNode `json:"nodes"`
	Edges []ServiceEdge `json:"edges"`
}
