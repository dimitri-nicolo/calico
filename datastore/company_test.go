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
		dsn := os.Getenv("TEST_DSN")
		if dsn == "" {
			t.Skip("TestInsertGetDeleteCompany being skipped due to a missing 'TEST_DSN' env variable")
		}
		db, _ := NewDB(dsn)
		c, err := db.CreateCompany(&Company{Key: testKey, Name: testName})
		if err != nil {
			t.Fatalf("error creating company: %s", err.Error())
		}

		id := c.Id
		uuid := c.Uuid

		require.Equal(t, testKey, c.Key)
		require.Equal(t, testName, c.Name)

		c, err = db.GetCompanyById(c.Id)
		if err != nil {
			t.Fatalf("error getting company: %s", err.Error())
		}
		require.Equal(t, id, c.Id)
		require.Equal(t, uuid, c.Uuid)
		require.Equal(t, testKey, c.Key)
		require.Equal(t, testName, c.Name)

		err = db.DeleteCompanyById(c.Id)
		if err != nil {
			t.Fatalf("error deleting company: %s", err.Error())
		}

		c, err = db.GetCompanyByUuid(c.Uuid)
		if err == nil {
			t.Fatalf("company should not exist, but it does")
		}
	})
}

func TestInsertCompanyUsingMock(t *testing.T) {
	t.Run("Test Insert Company Using an SQL mock", func(t *testing.T) {
		dbm, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
		}
		defer dbm.Close()
		db := &DB{dbm}

		uuid := uuidgen.NewV4().String()
		mock.ExpectExec("INSERT INTO companies").
			WithArgs(uuid, testKey, testName).
			WillReturnResult(sqlmock.NewResult(1, 1))

		c, err := db.CreateCompany(&Company{Uuid: uuid, Key: testKey, Name: testName})
		if err != nil {
			t.Errorf("error creating company")
		}

		require.Equal(t, int64(1), c.Id)
		require.Equal(t, uuid, c.Uuid)
		require.Equal(t, testKey, c.Key)
		require.Equal(t, testName, c.Name)

		// we make sure that all expectations were met
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})
}

func TestAllCompaniesUsingMock(t *testing.T) {
	t.Run("Test All Companies APIs using an SQL mock", func(t *testing.T) {
		dbm, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
		}
		defer dbm.Close()
		db := &DB{dbm}

		uuid := uuidgen.NewV4().String()
		rows := sqlmock.NewRows([]string{"id", "uuid", "ckey", "name"}).
			AddRow(int64(1), uuid, testKey, testName)

		mock.ExpectQuery("SELECT").
			WillReturnRows(rows)

		ca, err := db.AllCompanies()
		if err != nil {
			t.Errorf("error querying companies")
		}

		for _, c := range ca {
			require.Equal(t, int64(1), c.Id)
			require.Equal(t, uuid, c.Uuid)
			require.Equal(t, testKey, c.Key)
			require.Equal(t, testName, c.Name)
		}
		require.Equal(t, 1, len(ca))

		// we make sure that all expectations were met
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})
}
