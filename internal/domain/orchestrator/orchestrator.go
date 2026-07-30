package orchestrator

import (
	"fmt"
	"time"

	"github.com/y08lin4/proxy-cat/internal/autostable"
	"github.com/y08lin4/proxy-cat/internal/domain/proxy"
)

// ChainNode represents one hop in a dialer-proxy chain.
type ChainNode struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	Server     string `json:"server"`
	Port       int    `json:"port"`
	LatencyMS  int    `json:"latencyMs,omitempty"`
	Alive      bool   `json:"alive"`
	DialerName string `json:"dialerName,omitempty"`
}

// ChainTrace is a resolved dialer-proxy chain from entry to exit.
type ChainTrace struct {
	TargetNode string      `json:"targetNode"`
	Hops       []ChainNode `json:"hops"`
	TotalHops  int         `json:"totalHops"`
}

// Orchestrator validates and traces dialer-proxy chains.
type Orchestrator struct {
	manager *autostable.Manager
}

// New creates a new Orchestrator.
func New(manager *autostable.Manager) *Orchestrator {
	return &Orchestrator{manager: manager}
}

// ResolveChain follows the dialer-proxy chain for a given node name.
// Returns the chain from dialer to exit node.
func (o *Orchestrator) ResolveChain(p proxy.Profile, nodeName string) (*ChainTrace, error) {
	nodeMap := makeNodeMap(p)

	node, ok := nodeMap[nodeName]
	if !ok {
		return nil, fmt.Errorf("node %q not found", nodeName)
	}

	var hops []ChainNode
	current := node

	// Follow dialer-proxy chain, max 5 hops to prevent infinite loops
	for i := 0; i < 5; i++ {
		hop := makeChainNode(current)
		hops = append(hops, hop)

		dialerName := getDialerProxy(current)
		if dialerName == "" {
			break // End of chain
		}

		dialer, ok := nodeMap[dialerName]
		if !ok {
			hop.DialerName = dialerName // Mark missing dialer
			break
		}
		current = dialer
	}

	return &ChainTrace{
		TargetNode: nodeName,
		Hops:       hops,
		TotalHops:  len(hops),
	}, nil
}

// ValidateDialerChains checks all nodes with dialer-proxy references
// and ensures the referenced dialer nodes exist in the profile.
func (o *Orchestrator) ValidateDialerChains(p proxy.Profile) []DialerValidation {
	nodeMap := makeNodeMap(p)
	var results []DialerValidation

	for _, node := range p.Proxies {
		dialerName := getDialerProxy(node)
		if dialerName == "" {
			continue
		}
		_, ok := nodeMap[dialerName]
		results = append(results, DialerValidation{
			NodeName:   node.Name,
			DialerName: dialerName,
			Valid:      ok,
		})
	}
	return results
}

// DialerValidation reports the health of a single dialer-proxy reference.
type DialerValidation struct {
	NodeName   string `json:"nodeName"`
	DialerName string `json:"dialerName"`
	Valid      bool   `json:"valid"`
}

// GetDialerStatus returns the health status of all nodes in dialer chains.
func (o *Orchestrator) GetDialerStatus(p proxy.Profile) []DialerStatus {
	nodeMap := makeNodeMap(p)
	now := time.Now()
	snapshots := make(map[string]autostable.NodeSnapshot)
	for _, s := range o.manager.Snapshots(now) {
		snapshots[s.NodeID] = s
	}

	var statuses []DialerStatus
	for _, node := range p.Proxies {
		dialer := getDialerProxy(node)
		snapshot, hasSnapshot := snapshots[node.Name]
		alive := hasSnapshot && snapshot.Available

		statuses = append(statuses, DialerStatus{
			NodeName:   node.Name,
			DialerName: dialer,
			Alive:      alive,
			LatencyMS:  int(snapshot.LatencyMS),
		})

		// Also report dialer node status
		if dialer != "" {
			if dialerNode, ok := nodeMap[dialer]; ok {
				dSnap, dOk := snapshots[dialer]
				statuses = append(statuses, DialerStatus{
					NodeName:   dialerNode.Name,
					DialerName: "",
					Alive:      dOk && dSnap.Available,
					LatencyMS:  int(dSnap.LatencyMS),
					IsDialer:   true,
					ServesNode: node.Name,
				})
			}
		}
	}
	return statuses
}

// DialerStatus describes the health of one node in a dialer chain.
type DialerStatus struct {
	NodeName   string `json:"nodeName"`
	DialerName string `json:"dialerName,omitempty"`
	Alive      bool   `json:"alive"`
	LatencyMS  int    `json:"latencyMs,omitempty"`
	IsDialer   bool   `json:"isDialer,omitempty"`
	ServesNode string `json:"servesNode,omitempty"`
}

// makeNodeMap builds a name-indexed map of proxy nodes.
func makeNodeMap(p proxy.Profile) map[string]proxy.ProxyNode {
	m := make(map[string]proxy.ProxyNode, len(p.Proxies))
	for _, node := range p.Proxies {
		m[node.Name] = node
	}
	return m
}

// getDialerProxy returns the dialer-proxy value from a node's RawOptions, if any.
func getDialerProxy(node proxy.ProxyNode) string {
	if node.RawOptions == nil {
		return ""
	}
	v, ok := node.RawOptions["dialer-proxy"]
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return s
}

// makeChainNode converts a proxy.ProxyNode to a ChainNode.
func makeChainNode(node proxy.ProxyNode) ChainNode {
	return ChainNode{
		Name:       node.Name,
		Type:       node.Type,
		Server:     node.Server,
		Port:       node.Port,
		DialerName: getDialerProxy(node),
	}
}
