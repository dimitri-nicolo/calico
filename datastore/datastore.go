package datastore

import (
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
)

type Datastore interface {
	AllCompanies() ([]*Company, error)
	GetCompanyById(id int) (*Company, error)
	GetCompanyByUuid(uuid string) (*Company, error)
	CreateCompany(company *Company) (*Company, error)
	AllLicenses(companyId int) ([]*License, error)
	GetLicenseById(id int) (*License, error)
	CreateLicense(license *License) (*License, error)
}

type DB struct {
	*sql.DB
}

func NewDB(dataSourceName string) (*DB, error) {
	db, err := sql.Open("mysql", dataSourceName)
	if err != nil {
		return nil, err
	}
	if err = db.Ping(); err != nil {
		return nil, err
	}
	return &DB{db}, nil
}
