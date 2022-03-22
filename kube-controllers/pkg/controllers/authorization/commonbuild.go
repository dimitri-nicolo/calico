package authorization

import (
	"fmt"
	"strings"

	"github.com/projectcalico/calico/kube-controllers/pkg/elasticsearch"
)

func (n *nativeUserSynchronizer) eeResync() error {
	users, err := n.esCLI.GetUsers()
	if err != nil {
		return err
	}

	for _, user := range users {
		if strings.HasPrefix(user.Username, nativeUserPrefix) {
			subjectID := strings.TrimPrefix(user.Username, fmt.Sprintf("%s-", nativeUserPrefix))
			if !n.userCache.Exists(subjectID) {
				if err := n.esCLI.DeleteUser(elasticsearch.User{Username: user.Username}); err != nil {
					return err
				}
			}
		}
	}

	subjects := n.userCache.SubjectIDs()
	return n.synchronizeOIDCUsers(subjects)
}

func (n *nativeUserSynchronizer) eeDeleteEsUsers(esUsers map[string]elasticsearch.User) error {
	for _, esUser := range esUsers {
		if err := n.esCLI.DeleteUser(esUser); err != nil {
			return err
		}
	}
	return nil
}
