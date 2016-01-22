// Copyright 2016 CoreOS, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.package recipe
package integration

import (
	"testing"
	"time"

	"github.com/coreos/etcd/Godeps/_workspace/src/google.golang.org/grpc"
	"github.com/coreos/etcd/contrib/recipes"
)

func TestBarrierSingleNode(t *testing.T) {
	clus := newClusterGRPC(t, &clusterConfig{size: 3})
	defer clus.Terminate(t)
	testBarrier(t, 5, func() *grpc.ClientConn { return clus.conns[0] })
}

func TestBarrierMultiNode(t *testing.T) {
	clus := newClusterGRPC(t, &clusterConfig{size: 3})
	defer clus.Terminate(t)
	testBarrier(t, 5, func() *grpc.ClientConn { return clus.RandConn() })
}

func testBarrier(t *testing.T, waiters int, chooseConn func() *grpc.ClientConn) {
	b := recipe.NewBarrier(recipe.NewEtcdClient(chooseConn()), "test-barrier")
	if err := b.Hold(); err != nil {
		t.Fatalf("could not hold barrier (%v)", err)
	}
	if err := b.Hold(); err == nil {
		t.Fatalf("able to double-hold barrier")
	}

	donec := make(chan struct{})
	for i := 0; i < waiters; i++ {
		go func() {
			br := recipe.NewBarrier(recipe.NewEtcdClient(chooseConn()), "test-barrier")
			if err := br.Wait(); err != nil {
				t.Fatalf("could not wait on barrier (%v)", err)
			}
			donec <- struct{}{}
		}()
	}

	select {
	case <-donec:
		t.Fatalf("barrier did not wait")
	default:
	}

	if err := b.Release(); err != nil {
		t.Fatalf("could not release barrier (%v)", err)
	}

	timerC := time.After(time.Duration(waiters*100) * time.Millisecond)
	for i := 0; i < waiters; i++ {
		select {
		case <-timerC:
			t.Fatalf("barrier timed out")
		case <-donec:
		}
	}
}
