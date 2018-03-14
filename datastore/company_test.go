package datastore

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	uuidgen "github.com/satori/go.uuid"
	"github.com/stretchr/testify/require"
)

func TestInsertCompany(t *testing.T) {
	t.Run("Test Insert Company", func(t *testing.T) {
		const (
			testKey = "TestCompanyKey"
			testName = "Test Company Name"
		)
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

func TestAllCompanies(t *testing.T) {
	t.Run("Test All Companies APIs", func(t *testing.T) {
		const (
			testKey = "TestCompanyKey"
			testName = "Test Company Name"
		)
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
