package datastore

import (
	"os"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	uuidgen "github.com/satori/go.uuid"
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

		c, err := db.CreateCompany(&Company{Key: testKey, Name: testName})
		req.Nil(err, "error creating company: %s", err)
		req.NotNil(c, "company reference is nil!")

		id := c.Id
		uuid := c.Uuid

		req.Equal(testKey, c.Key)
		req.Equal(testName, c.Name)

		c, err = db.GetCompanyById(c.Id)
		req.Nil(err, "error getting company: %s", err)
		req.NotNil(c, "company reference is nil!")

		req.Equal(id, c.Id)
		req.Equal(uuid, c.Uuid)
		req.Equal(testKey, c.Key)
		req.Equal(testName, c.Name)

		c, err = db.GetCompanyByUuid(c.Uuid)
		req.Nil(err, "error getting company: %s", err)
		req.NotNil(c, "company reference is nil!")

		req.Equal(id, c.Id)
		req.Equal(uuid, c.Uuid)
		req.Equal(testKey, c.Key)
		req.Equal(testName, c.Name)

		err = db.DeleteCompanyById(c.Id)
		req.Nil(err, "error deleting company: %s", err)

		c, err = db.GetCompanyByUuid(c.Uuid)
		req.NotNil(err, "company should not exist, but it does")
		req.Nil(c, "company should not exist, but it does")
	})
}

func TestInsertCompanyUsingMock(t *testing.T) {
	t.Run("Test Insert Company Using an SQL mock", func(t *testing.T) {
		req := require.New(t)
		dbm, mock, err := sqlmock.New()
		req.Nil(err, "error occurred when creating mock db:  %s", err)
		defer dbm.Close()
		db := &DB{dbm}

		uuid := uuidgen.NewV4().String()
		mock.ExpectExec("INSERT INTO companies").
			WithArgs(uuid, testKey, testName).
			WillReturnResult(sqlmock.NewResult(1, 1))

		c, err := db.CreateCompany(&Company{Uuid: uuid, Key: testKey, Name: testName})
		req.Nil(err, "error occurred when creating company:  %s", err)
		req.NotNil(c, "company reference cannot be nil")

		req.Equal(int64(1), c.Id)
		req.Equal(uuid, c.Uuid)
		req.Equal(testKey, c.Key)
		req.Equal(testName, c.Name)

		// we make sure that all expectations were met
		err = mock.ExpectationsWereMet()
		req.Nil(err, "there were unfulfilled expectations: %s", err)
	})
}

func TestAllCompaniesUsingMock(t *testing.T) {
	t.Run("Test All Companies APIs using an SQL mock", func(t *testing.T) {
		req := require.New(t)
		dbm, mock, err := sqlmock.New()
		req.Nil(err, "error occurred when creating mock db:  %s", err)
		defer dbm.Close()
		db := &DB{dbm}

		uuid := uuidgen.NewV4().String()
		rows := sqlmock.NewRows([]string{"id", "uuid", "ckey", "name"}).
			AddRow(int64(1), uuid, testKey, testName)

		mock.ExpectQuery("SELECT").
			WillReturnRows(rows)

		ca, err := db.AllCompanies()
		req.Nil(err, "error occurred when getting all companies:  %s", err)
		req.NotNil(ca, "company reference cannot be nil")

		for _, c := range ca {
			req.Equal(int64(1), c.Id)
			req.Equal(uuid, c.Uuid)
			req.Equal(testKey, c.Key)
			req.Equal(testName, c.Name)
		}
		req.Equal(1, len(ca))

		// we make sure that all expectations were met
		err = mock.ExpectationsWereMet()
		req.Nil(err, "there were unfulfilled expectations: %s", err)
	})
}
