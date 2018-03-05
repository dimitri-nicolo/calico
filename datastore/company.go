package datastore

import (
	"github.com/satori/go.uuid"
)

type Company struct {
	Id int
	Uuid string
	Name string
	Key string
}

func (db *DB) AllCompanies() ([]*Company, error) {
	rows, err := db.Query("SELECT id, uuid, key, name FROM companies")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	companies := make([]*Company, 0)
	for rows.Next() {
		cmp := &Company{}
		err := rows.Scan(&cmp.Id, &cmp.Uuid, &cmp.Key, &cmp.Name)
		if err != nil {
			return nil, err
		}
		companies = append(companies, cmp)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return companies, nil
}

func (db *DB) GetCompanyById(id int) (*Company, error) {
	cmp := &Company{}
	row := db.QueryRow("SELECT id, uuid, key, name FROM companies WHERE id = ?", id)
	err := row.Scan(&cmp.Id, &cmp.Uuid, &cmp.Key, &cmp.Name)
	if err != nil {
		return nil, err
	}
	return cmp, nil
}

func (db *DB) GetCompanyByUuid(uuid string) (*Company, error) {
	cmp := &Company{}
	row := db.QueryRow("SELECT id, uuid, key, name FROM companies WHERE uuid = ?", uuid)
	err := row.Scan(&cmp.Id, &cmp.Uuid, &cmp.Key, &cmp.Name)
	if err != nil {
		return nil, err
	}
	return cmp, nil
}

func (db *DB) CreateCompany(company *Company) (*Company, error) {
	if company.Uuid == "" {
		company.Uuid = uuid.NewV4().String()
	}
	err := db.QueryRow("INSERT INTO company (uuid, key, name) VALUES (?, ?, ?); SELECT LAST_INSERT_ID();",
		company.Uuid, company.Key, company.Name).Scan(&company.Id)
	if err != nil {
		return nil, err
	}
	return company, nil
}
