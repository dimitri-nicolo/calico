package datastore

import (
	"database/sql"

	_ "github.com/go-sql-driver/mysql"

	api "github.com/projectcalico/libcalico-go/lib/apis/v3"
	"github.com/tigera/licensing/client"
)

var DSN string = "tigera_carrotctl:JbUEMjuHqVpyCCjt@/tigera_backoffice"

type Datastore interface {
	AllCompanies() ([]*Company, error)
	GetCompanyByName(name string) (int64, error)
	GetCompanyById(id int) (*Company, error)
	GetCompanyByUuid(uuid string) (*Company, error)
	CreateCompany(name string) (int64, error)
	DeleteCompanyById(id int64) error

	//AllLicenses(companyId int) ([]*License, error)
	//GetLicenseById(id int) (*License, error)
	CreateLicense(license *api.LicenseKey, companyID int, claims *client.LicenseClaims) (int64, error)
	DeleteLicense(licenseID int64) error
}

type DB struct {
	*sql.DB
}

func NewDB(dsn string) (*DB, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	if err = db.Ping(); err != nil {
		return nil, err
	}
	return &DB{db}, nil
}
