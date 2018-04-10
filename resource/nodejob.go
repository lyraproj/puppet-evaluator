package resource

import (
	"github.com/puppetlabs/go-evaluator/eval"
	"sync/atomic"
)

type nodeJob struct {
	n *node
	c eval.Context
}

type jobCounter struct {
	counter int32
}

func (jc *jobCounter) decrement() int32 {
	return atomic.AddInt32(&jc.counter, -1)
}

func (jc *jobCounter) increment() int32 {
	return atomic.AddInt32(&jc.counter, 1)
}

// This function represents a worker that resolves nodes
func nodeWorker(id int, nodeJobs <-chan *nodeJob, done chan<- bool) {
	for nj := range nodeJobs {
		func(job *nodeJob) {
			defer func() {
				if getJobCounter(job.c).decrement() == 0 {
					// Last node job done. Close nodeJobs channel
					done <- true
					close(getNodeJobs(job.c))
				}
			}()
			job.n.evaluate(nj.c)
		}(nj)
	}
}

// Schedule the given nodes for evaluation
func scheduleNodes(c eval.Context, nodes eval.IndexedValue) {
	if nodes.Len() > 0 {
		jc := getJobCounter(c)
		nodeJobs := getNodeJobs(c)
		nodes.Each(func(ch eval.PValue) {
			jc.increment()
			nodeJobs <- &nodeJob{n: ch.(*node), c: c.Fork()}
		})
	}
}
