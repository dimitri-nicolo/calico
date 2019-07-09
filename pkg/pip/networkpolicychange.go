package pip

import (
	"bytes"
	"encoding/json"
	"fmt"

	v3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
	log "github.com/sirupsen/logrus"
	"github.com/tigera/compliance/pkg/resources"

	v1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type NetworkPolicyChange struct {
	ChangeAction  string
	NetworkPolicy resources.Resource
}

func (c *NetworkPolicyChange) UnmarshalJSON(b []byte) error {
	log.Info("Unmarshal NPC")

	var m map[string]interface{}
	err := json.Unmarshal(b, &m)
	if err != nil {
		return err
	}

	//get the change action
	ca, ok := m["action"]
	if !ok {
		log.Error("Network policy change action not found")
		return fmt.Errorf("Network policy change action not found")
	}
	c.ChangeAction = ca.(string)

	//drill down to the policy
	pol, ok := m["policy"]
	if !ok {
		log.Error("Network policy change policy not found")
		return fmt.Errorf("Network policy change policy not found")
	}

	polb, err := json.Marshal(pol)
	if err != nil {
		return err
	}

	var tm metav1.TypeMeta
	err = json.Unmarshal(polb, &tm)
	if err != nil {
		return err
	}

	if tm.APIVersion == "" || tm.Kind == "" {
		log.WithField("apiVersion", tm.APIVersion).WithField("kind", tm.Kind).Error("Could not identify resource type")
		return fmt.Errorf("Could not identify resource type")
	}

	// figure out what kind of policy it is
	if tm.APIVersion == "projectcalico.org/v3" && tm.Kind == v3.KindNetworkPolicy {
		//v3.NetworkPolicy
		log.Info("calico v3.NetworkPolicy found")
		var np v3.NetworkPolicy
		err := json.Unmarshal(polb, &np)
		if err != nil {
			return err
		}
		log.Debug("Calico Network Policy:", np)
		c.NetworkPolicy = &np
	} else if tm.APIVersion == "projectcalico.org/v3" && tm.Kind == v3.KindGlobalNetworkPolicy {
		//v3.GlobalNetworkPolicy
		log.Debug("calico v3.GlobalNetworkPolicy found")
		var np v3.GlobalNetworkPolicy
		err := json.Unmarshal(polb, &np)
		if err != nil {
			return err
		}
		log.Debug("Global Network Policy", np)
		c.NetworkPolicy = &np
	} else if tm.APIVersion == "networking.k8s.io/v1" && tm.Kind == "NetworkPolicy" {
		// v1.NetworkPolicy
		log.Debug("k8s v1.NetworkPolicy policy found")
		var np v1.NetworkPolicy
		err := json.Unmarshal(polb, &np)
		if err != nil {
			return err
		}
		log.Debug("K8s Network Policy", np)
		c.NetworkPolicy = &np
	} else {
		return fmt.Errorf("Unknown policy type")
	}

	return nil
}

func (c *NetworkPolicyChange) MarshalJSON() ([]byte, error) {
	log.Info("Marshal NPC")

	b, err := json.Marshal(c.NetworkPolicy)

	if err != nil {
		return b, nil
	}

	//wrap it back in the policy
	buffer := bytes.NewBufferString("{ \"policy\": ")
	buffer.Write(b)
	buffer.WriteString(fmt.Sprintf(",\"action\":\"%v\"}", c.ChangeAction))

	log.Info("FULLSTRING", string(buffer.Bytes()))

	return buffer.Bytes(), nil

}
