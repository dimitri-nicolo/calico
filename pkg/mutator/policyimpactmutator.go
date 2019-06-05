// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package mutator

import (
	"bytes"
	"context"
	"io/ioutil"
	"net/http"

	log "github.com/sirupsen/logrus"
	"github.com/tigera/es-proxy/pkg/pip"
	"github.com/tigera/es-proxy/pkg/pip/flow"
)

type ResponseHook interface {
	ModifyResponse(*http.Response) error
}

type responseHook struct {
	pip pip.PIP
}

func NewResponseHook(p pip.PIP) ResponseHook {
	return &responseHook{
		pip: p,
	}
}

// ModifyResponse alters the flows in the response by calling the
// CalculateFlowImpact method of the PIP object with the extracted flow data
func (rh *responseHook) ModifyResponse(r *http.Response) error {
	log.Info("PIP response hook")

	//TODO: get these from the response.request if it's there. Otherwise store it
	// somewhere when it comes in with the request and then retrieve it here
	changes := make([]pip.NetworkPolicyChange, 0)

	context := context.TODO()

	//read the flows from the response body
	b, err := ioutil.ReadAll(r.Body)

	if err != nil {
		return err
	}

	//create a flow manager and unmarshal the data
	v := flow.NewFlowManager()
	err = v.Unmarshal(b)

	if err != nil {
		return err
	}

	//if there are no flows, there is no error but we are done
	if !v.HasFlows() {
		return nil
	}

	//extract the flows
	inflows, err := v.ExtractFlows()
	if err != nil {
		return err
	}

	//calculate the flow impact
	outflows, _ := rh.pip.CalculateFlowImpact(context, changes, inflows)
	//TODO: check the error from above

	//put the returned flows back into the response body and remarshal
	v.ReplaceFlows(outflows)
	newBodyContent, err := v.Marshal()

	if err != nil {
		return err
	}
	r.Body = ioutil.NopCloser(bytes.NewReader(newBodyContent))

	// fix the content length as it might have changed
	r.ContentLength = int64(len(newBodyContent))

	return nil
}
