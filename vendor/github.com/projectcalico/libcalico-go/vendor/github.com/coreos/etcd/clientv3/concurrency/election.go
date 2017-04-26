// Copyright 2016 The etcd Authors
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
// limitations under the License.

package concurrency

import (
	"errors"
	"fmt"

	v3 "github.com/coreos/etcd/clientv3"
	pb "github.com/coreos/etcd/etcdserver/etcdserverpb"
	"github.com/coreos/etcd/mvcc/mvccpb"
	"golang.org/x/net/context"
)

var (
	ErrElectionNotLeader = errors.New("election: not leader")
	ErrElectionNoLeader  = errors.New("election: no leader")
)

type Election struct {
	session *Session

	keyPrefix string

	leaderKey     string
	leaderRev     int64
	leaderSession *Session
	hdr           *pb.ResponseHeader
}

// NewElection returns a new election on a given key prefix.
func NewElection(s *Session, pfx string) *Election {
	return &Election{session: s, keyPrefix: pfx + "/"}
}

// ResumeElection initializes an election with a known leader.
func ResumeElection(s *Session, pfx string, leaderKey string, leaderRev int64) *Election {
	return &Election{
		session:       s,
		leaderKey:     leaderKey,
		leaderRev:     leaderRev,
		leaderSession: s,
	}
}

// Campaign puts a value as eligible for the election. It blocks until
// it is elected, an error occurs, or the context is cancelled.
func (e *Election) Campaign(ctx context.Context, val string) error {
	s := e.session
	client := e.session.Client()

	k := fmt.Sprintf("%s%x", e.keyPrefix, s.Lease())
	txn := client.Txn(ctx).If(v3.Compare(v3.CreateRevision(k), "=", 0))
	txn = txn.Then(v3.OpPut(k, val, v3.WithLease(s.Lease())))
	txn = txn.Else(v3.OpGet(k))
	resp, err := txn.Commit()
	if err != nil {
		return err
	}
	e.leaderKey, e.leaderRev, e.leaderSession = k, resp.Header.Revision, s
	if !resp.Succeeded {
		kv := resp.Responses[0].GetResponseRange().Kvs[0]
		e.leaderRev = kv.CreateRevision
		if string(kv.Value) != val {
			if err = e.Proclaim(ctx, val); err != nil {
				e.Resign(ctx)
				return err
			}
		}
	}

	_, err = waitDeletes(ctx, client, e.keyPrefix, e.leaderRev-1)
	if err != nil {
		// clean up in case of context cancel
		select {
		case <-ctx.Done():
			e.Resign(client.Ctx())
		default:
			e.leaderSession = nil
		}
		return err
	}
	e.hdr = resp.Header

	return nil
}

// Proclaim lets the leader announce a new value without another election.
func (e *Election) Proclaim(ctx context.Context, val string) error {
	if e.leaderSession == nil {
		return ErrElectionNotLeader
	}
	client := e.session.Client()
	cmp := v3.Compare(v3.CreateRevision(e.leaderKey), "=", e.leaderRev)
	txn := client.Txn(ctx).If(cmp)
	txn = txn.Then(v3.OpPut(e.leaderKey, val, v3.WithLease(e.leaderSession.Lease())))
	tresp, terr := txn.Commit()
	if terr != nil {
		return terr
	}
	if !tresp.Succeeded {
		e.leaderKey = ""
		return ErrElectionNotLeader
	}

	e.hdr = tresp.Header
	return nil
}

// Resign lets a leader start a new election.
func (e *Election) Resign(ctx context.Context) (err error) {
	if e.leaderSession == nil {
		return nil
	}
	client := e.session.Client()
	cmp := v3.Compare(v3.CreateRevision(e.leaderKey), "=", e.leaderRev)
	resp, err := client.Txn(ctx).If(cmp).Then(v3.OpDelete(e.leaderKey)).Commit()
	if err == nil {
		e.hdr = resp.Header
	}
	e.leaderKey = ""
	e.leaderSession = nil
	return err
}

// Leader returns the leader value for the current election.
func (e *Election) Leader(ctx context.Context) (*v3.GetResponse, error) {
	client := e.session.Client()
	resp, err := client.Get(ctx, e.keyPrefix, v3.WithFirstCreate()...)
	if err != nil {
		return nil, err
	} else if len(resp.Kvs) == 0 {
		// no leader currently elected
		return nil, ErrElectionNoLeader
	}
	return resp, nil
}

// Observe returns a channel that reliably observes ordered leader proposals
// as GetResponse values on every current elected leader key. It will not
// necessarily fetch all historical leader updates, but will always post the
// most recent leader value.
//
// The channel closes when the context is canceled or the underlying watcher
// is otherwise disrupted.
func (e *Election) Observe(ctx context.Context) <-chan v3.GetResponse {
	retc := make(chan v3.GetResponse)
	go e.observe(ctx, retc)
	return retc
}

func (e *Election) observe(ctx context.Context, ch chan<- v3.GetResponse) {
	client := e.session.Client()

	defer close(ch)
	lastRev := int64(0)
	for {
		opts := append(v3.WithFirstCreate(), v3.WithRev(lastRev))
		resp, err := client.Get(ctx, e.keyPrefix, opts...)
		if err != nil {
			return
		}

		var kv *mvccpb.KeyValue

		if len(resp.Kvs) == 0 {
			cctx, cancel := context.WithCancel(ctx)
			// wait for first key put on prefix
			opts := []v3.OpOption{v3.WithRev(resp.Header.Revision), v3.WithPrefix()}
			wch := client.Watch(cctx, e.keyPrefix, opts...)
			for kv == nil {
				wr, ok := <-wch
				if !ok || wr.Err() != nil {
					cancel()
					return
				}
				// only accept PUTs; a DELETE will make observe() spin
				for _, ev := range wr.Events {
					if ev.Type == mvccpb.PUT {
						kv = ev.Kv
						break
					}
				}
			}
			cancel()
		} else {
			kv = resp.Kvs[0]
		}

		cctx, cancel := context.WithCancel(ctx)
		wch := client.Watch(cctx, string(kv.Key), v3.WithRev(kv.ModRevision))
		keyDeleted := false
		for !keyDeleted {
			wr, ok := <-wch
			if !ok {
				return
			}
			for _, ev := range wr.Events {
				if ev.Type == mvccpb.DELETE {
					lastRev = ev.Kv.ModRevision
					keyDeleted = true
					break
				}
				resp.Header = &wr.Header
				resp.Kvs = []*mvccpb.KeyValue{ev.Kv}
				select {
				case ch <- *resp:
				case <-cctx.Done():
					return
				}
			}
		}
		cancel()
	}
}

// Key returns the leader key if elected, empty string otherwise.
func (e *Election) Key() string { return e.leaderKey }

// Rev returns the leader key's creation revision, if elected.
func (e *Election) Rev() int64 { return e.leaderRev }

// Header is the response header from the last successful election proposal.
func (m *Election) Header() *pb.ResponseHeader { return m.hdr }
