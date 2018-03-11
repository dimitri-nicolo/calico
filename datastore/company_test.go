package datastore

import (
	"fmt"
	"testing"

	"github.com/onsi/gomega"
)

func TestAllCompanies(t *testing.T) {
	t.Run("Test All Companies", func(t *testing.T) {
		gomega.RegisterTestingT(t)
		db, err := NewDB(fmt.Sprintf("%v:%v@/tigera_backoffice", "root", "r00tPa$$w0rd"))
		defer tearDown()
		if err != nil {
			t.Errorf("unable to connect to the db for testing")
		}
		c, err := db.CreateCompany(&Company{Key: "TestKey", Name: "Testing Company"})
		if err != nil {
			t.Errorf("error creating company")
		}
		c, err = db.GetCompanyById(c.Id)
		if err != nil {
			t.Errorf("error getting company by id")
		}
		c, err = db.GetCompanyByUuid(c.Uuid)
		if err != nil {
			t.Errorf("error getting company by uuid")
		}

	})
}

func tearDown() {

}
