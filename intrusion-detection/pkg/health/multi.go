// Copyright 2019 Tigera Inc. All rights reserved.

package health

import "context"

type Pingers []Pinger

func (pingers Pingers) Ping(ctx context.Context) error {
	for _, p := range pingers {
		if err := p.Ping(ctx); err != nil {
			return err
		}
	}
	return nil
}

type Readiers []Readier

func (readiers Readiers) Ready() bool {
	for _, r := range readiers {
		if !r.Ready() {
			return false
		}
	}
	return true
}
