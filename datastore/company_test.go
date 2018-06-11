package datastore

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	testKey = "TestCompanyKey"
	testName = "Test Company Name"
)

func TestInsertGetDeleteCompany(t *testing.T) {
	t.Run("Test Insert Company", func(t *testing.T) {
		req := require.New(t)
		dsn := os.Getenv("TEST_DSN")
		if dsn == "" {
			t.Skip("TestInsertGetDeleteCompany being skipped due to a missing 'TEST_DSN' env variable")
		}

		db, err := NewDB(dsn)
		req.Nil(err, "error connecting to database: %s", err)
		req.NotNil(db, "db reference is nil!")

		id, err := db.CreateCompany(testName)
		req.Nil(err, "error creating company: %s", err)

		cid, err := db.GetCompanyByName(testName)
		req.Nil(err, "error getting company: %s", err)
		req.Equal(id, cid)

		c, err := db.GetCompanyById(id)
		req.Nil(err, "error getting company: %s", err)
		req.NotNil(c, "company reference is nil!")
		req.Equal(testName, c.Name)
		req.Equal(id, c.ID)

		err = db.DeleteCompanyById(id)
		req.Nil(err, "error deleting company: %s", err)

		c, err = db.GetCompanyById(id)
		req.NotNil(err, "company should not exist, but it does")
		req.Nil(c, "company should not exist, but it does")
	})
}

