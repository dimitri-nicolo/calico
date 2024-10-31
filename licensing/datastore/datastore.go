package datastore

import (
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
	"github.com/kelseyhightower/envconfig"
	api "github.com/tigera/api/pkg/apis/projectcalico/v3"

	"github.com/projectcalico/calico/licensing/client"
)

var DSN string

type DBAccess struct {
	User     string `default:"tigera_carrotctl"`
	Password string `default:"JbUEMjuHqVpyCCjt"`
	DNS      string `default:"localhost"`
	Port     string `default:"3306"`
	Name     string `default:"tigera_backoffice"`
}

func init() {
	// Parse env variables to get DB access information.
	var db DBAccess
	_ = envconfig.Process("carrotctl", &db)

	DSN = fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true", db.User, db.Password, db.DNS, db.Port, db.Name)
}

type Datastore interface {
	AllCompanies() ([]*Company, error)
	GetCompanyIdByName(name string) (int64, error)
	GetCompanyById(id int) (*Company, error)
	CreateCompany(name string) (int64, error)
	DeleteCompanyById(id int64) error

	GetLicenseByUUID(uuid string) (*LicenseInfo, error)
	GetLicensesByCompany(companyID int64) ([]*LicenseInfo, error)
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
