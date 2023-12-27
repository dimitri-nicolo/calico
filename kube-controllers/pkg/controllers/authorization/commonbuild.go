package authorization

import (
	"strings"

	"github.com/projectcalico/calico/kube-controllers/pkg/elasticsearch"
)

func (n *nativeUserSynchronizer) eeResync() error {
	users, err := n.esCLI.GetUsers()
	if err != nil {
		return err
	}

	for _, user := range users {
		if strings.HasPrefix(user.Username, n.esUserPrefix) {
			subjectID := strings.TrimPrefix(user.Username, n.esUserPrefix)
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
