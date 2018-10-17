package resource

import (
	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-evaluator/threadlocal"
	"sync/atomic"
)

type nodeJob struct {
	n *node
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
				threadlocal.Cleanup()
				if getJobCounter(job.n.Context()).decrement() == 0 {
					// Last node job done. Close nodeJobs channel
					done <- true
					close(getNodeJobs(job.n.Context()))
				}
			}()
			threadlocal.Init()
			threadlocal.Set(eval.PuppetContextKey, job.n.Context())
			job.n.evaluate()
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
			nd := ch.(*node)
			nodeJobs <- &nodeJob{n: nd}
		})
	}
}
