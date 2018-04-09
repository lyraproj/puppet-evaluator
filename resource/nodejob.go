package resource

import (
	"github.com/puppetlabs/go-evaluator/eval"
)

type(
	nodeJob struct {
		n *node
		c eval.Context
	}
)


// This function represents a worker that resolves nodes
func nodeWorker(nodeJobs <- chan *nodeJob) {
	for nj := range nodeJobs {
		nj.n.evaluate(nj.c)
	}
}

// Schedule the given nodes for evaluation
func scheduleNodes(c eval.Context, nodes eval.IndexedValue) {
	if nodes.Len() > 0 {
		nodeJobs := getNodeJobs(c)
		nodes.Each(func(ch eval.PValue) {
			nodeJobs <- &nodeJob{n: ch.(*node), c: c.Fork()}
		})
	}
}
