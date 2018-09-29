package node

import (
	"github.com/perlin-network/graph/wire"
	"github.com/perlin-network/noise/network/rpc"
	"github.com/pkg/errors"
	"sync"
	"time"
)

var (
	ErrPrecommit = errors.New("failed to precommit tx")
)

const (
	K     = 1
	Alpha = 0.8
)

type query struct {
	*Wavelet
	sybil
}

func (q query) Query(wired *wire.Transaction) error {
	addresses := q.routes.FindClosestPeers(q.net.ID, K+1)

	var wg sync.WaitGroup
	wg.Add(len(addresses))

	responses := make([]bool, len(addresses))

	for i, address := range addresses {
		go func(i int, address string) {
			defer wg.Done()

			client, err := q.net.Client(address)

			if err != nil {
				responses[i] = false
				return
			}

			request := new(rpc.Request)
			request.SetTimeout(10 * time.Second)
			request.SetMessage(wired)

			response, err := client.Request(request)
			if err != nil {
				responses[i] = false
				return
			}

			if response, ok := response.(*QueryResponse); ok && response.StronglyPreferred {
				responses[i] = true
				return
			}

			responses[i] = false
		}(i, address.Address)
	}

	wg.Wait()

	positives := q.weigh(addresses, responses, wired)

	if positives < Alpha {
		return errors.Wrapf(ErrPrecommit, "expected >= %.2f alpha; got only %.2f positives", Alpha, positives)
	}

	return nil
}