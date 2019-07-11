// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package mutator

import (
	"bytes"
	"io/ioutil"
	"net/http"

	log "github.com/sirupsen/logrus"
	"github.com/tigera/es-proxy/pkg/pip"
	"github.com/tigera/es-proxy/pkg/pip/flow"
)

type pipResponseHook struct {
	pip pip.PIP
}

func NewPIPResponseHook(p pip.PIP) ResponseHook {
	return &pipResponseHook{
		pip: p,
	}
}

// ModifyResponse alters the flows in the response by calling the
// CalculateFlowImpact method of the PIP object with the extracted flow data
func (rh *pipResponseHook) ModifyResponse(r *http.Response) error {

	//extract the context from the request
	cxt := r.Request.Context()

	//look for the policy impact request data in the context
	changes := cxt.Value(pip.PolicyImpactContextKey)

	//if there were no changes, no need to modify the response
	if changes == nil {
		return nil
	}

	log.Debug("Policy Impact ModifyResponse executing")

	//assert that we have network policy changes
	res := changes.([]pip.ResourceChange)

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
	/*
		inflows, err := v.ExtractFlows()
		if err != nil {
			return err
		}
	*/

	//calculate the flow impact
	pc, err := rh.pip.GetPolicyCalculator(cxt, res)
	if err != nil {
		return err
	}

	processed, before, after := pc.Action(nil)
	log.WithFields(log.Fields{
		"processed": processed,
		"before":    before,
		"after":     after,
	}).Debug("Processed flow")
	/*
		outflows, err := rh.pip.CalculateFlowImpact(cxt, res, inflows)
		if err != nil {
			return err
		}

		//put the returned flows back into the response body and remarshal
		v.ReplaceFlows(ouflows)
	*/
	newBodyContent, err := v.Marshal()
	if err != nil {
		return err
	}
	r.Body = ioutil.NopCloser(bytes.NewReader(newBodyContent))

	// fix the content length as it might have changed
	r.ContentLength = int64(len(newBodyContent))

	return nil
}
