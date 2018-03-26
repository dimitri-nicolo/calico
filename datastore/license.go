package datastore

import (
	"strings"

	api "github.com/projectcalico/libcalico-go/lib/apis/v3"
	"github.com/tigera/licensing/client"
)

/* Comment out for now: fix up when list and get are implemented.
func (db *DB) AllLicenses(companyId int) ([]*License, error) {
	rows, err := db.Query("SELECT id, jwt FROM licenses WHERE company_id = ?", companyId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	licenses := make([]*License, 0)
	for rows.Next() {
		lic := &License{}
		err := rows.Scan(&lic.Id, &lic.Jwt)
		if err != nil {
			return nil, err
		}
		lic.CompanyId = companyId
		licenses = append(licenses, lic)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return licenses, nil
}

func (db *DB) GetLicenseById(id int) (*License, error) {
	lic := &License{}
	row := db.QueryRow("SELECT id, jwt FROM licenses WHERE id = ?", id)
	err := row.Scan(&lic.Id, &lic.Jwt)
	if err != nil {
		return nil, err
	}
	return lic, nil
}
*/

// CreateLicense saves a license in the database; returning success and the licenseID.
func (db *DB) CreateLicense(license *api.LicenseKey, companyID int64, claims *client.LicenseClaims) (int64, error) {
	// Leave the following fields unset since they're not implemented yet:
	// - cluster_guid
	res, err := db.Exec("INSERT INTO licenses "+
		"(license_uuid, nodes, company_id, version, features, grace_period, checkin_int, expiry, issued_at, jwt) "+
		"VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		claims.LicenseID,
		claims.Nodes,
		companyID,
		claims.Version,
		strings.Join(claims.Features, "|"),
		claims.GracePeriod,
		claims.CheckinInterval,
		claims.Expiry.Time(),
		claims.IssuedAt.Time(),
		license.Spec.Token,
	)
	if err != nil {
		return 0, err
	}

	licenseID, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}

	return licenseID, nil
}

// DeleteLicense removes a license from the database, given the ID returned by CreateLicense().
func (db *DB) DeleteLicense(licenseID int64) error {
	_, err := db.Exec("DELETE FROM licenses WHERE id = ?", licenseID)
	if err != nil {
		return err
	}

	return nil
}
