// Copyright (c) 2019-2020 Tigera, Inc. All rights reserved.

// This package is responsible for the communicating with elasticsearch, mainly transferring objects to requests to send
// to elasticsearch and parsing the responses from elasticsearch
package elasticsearch

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	es7 "github.com/elastic/go-elasticsearch/v7"
)

type client struct {
	*es7.Client
}

type Client interface {
	GetUsers() ([]User, error)
	UserExists(username string) (bool, error)
	UpdateUser(user User) error
	DeleteUser(user User) error
	CreateUser(user User) error
	DeleteRole(role Role) error
}

// User represents an Elasticsearch user, which may or may not have roles attached to it
type User struct {
	Username string
	Password string
	Roles    []Role
}

// RoleNames is a convenience function for getting the names of all the roles defined for this Elasticsearch user
func (u User) RoleNames() []string {
	var names []string
	for _, role := range u.Roles {
		names = append(names, role.Name)
	}

	return names
}

// SecretName returns the name of the secret that should be used to store the information of this user
func (u User) SecretName() string {
	return fmt.Sprintf("%s-elasticsearch-access", u.Username)
}

// Role represents an Elasticsearch role that may be attached to a User
type Role struct {
	Name       string `json:"-"`
	Definition *RoleDefinition
}

type RoleDefinition struct {
	Cluster      []string      `json:"cluster"`
	Indices      []RoleIndex   `json:"indices"`
	Applications []Application `json:"applications,omitempty"`
}

type RoleIndex struct {
	Names      []string `json:"names"`
	Privileges []string `json:"privileges"`
}

type Application struct {
	Application string   `json:"application"`
	Privileges  []string `json:"privileges"`
	Resources   []string `json:"resources"`
}

func NewClient(url, username, password string, roots *x509.CertPool) (Client, error) {
	config := es7.Config{
		Addresses: []string{
			url,
		},
		Username: username,
		Password: password,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: roots,
			},
		},
	}

	esClient, err := es7.NewClient(config)
	if err != nil {
		return nil, err
	}

	return &client{esClient}, nil
}

// createRoles wraps createRoles to make creating multiple rows slightly more convenient
func (cli *client) createRoles(roles ...Role) error {
	for _, role := range roles {
		if err := cli.createRole(role); err != nil {
			return err
		}
	}

	return nil
}

// createRole attempts to create (or updated) the given Elasticsearch role.
func (cli *client) createRole(role Role) error {
	if role.Name == "" {
		return fmt.Errorf("can't create a role with an empty name")
	}

	j, err := json.Marshal(role.Definition)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("/_security/role/%s", role.Name), bytes.NewBuffer(j))
	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", "application/json")

	response, err := cli.Perform(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		body, err := ioutil.ReadAll(response.Body)
		if err != nil {
			return err
		}
		return fmt.Errorf(string(body))
	}

	return nil
}

// DeleteRole will delete the Elasticsearch role
func (cli *client) DeleteRole(role Role) error {
	if role.Name == "" {
		return fmt.Errorf("can't delete a role with an empty name")
	}

	req, err := http.NewRequest("DELETE", fmt.Sprintf("/_security/role/%s", role.Name), nil)
	if err != nil {
		return err
	}

	response, err := cli.Perform(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != 200 && response.StatusCode != 404 {
		body, err := ioutil.ReadAll(response.Body)
		if err != nil {
			return err
		}
		return fmt.Errorf(string(body))
	}

	return nil
}

// CreateUser will create the Elasticsearch user and roles (if any roles are defined for the user). If the roles exist they
// will be updated.
func (cli *client) CreateUser(user User) error {
	var rolesToCreate []Role
	for _, role := range user.Roles {
		if role.Definition != nil {
			rolesToCreate = append(rolesToCreate, role)
		}
	}

	if len(rolesToCreate) > 0 {
		if err := cli.createRoles(rolesToCreate...); err != nil {
			return err
		}
	}

	j, err := json.Marshal(map[string]interface{}{
		"password": user.Password,
		"roles":    user.RoleNames(),
	})
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("/_security/user/%s", user.Username), bytes.NewBuffer(j))
	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", "application/json")

	response, err := cli.Perform(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		body, err := ioutil.ReadAll(response.Body)
		if err != nil {
			return err
		}
		return fmt.Errorf(string(body))
	}

	return nil
}

// DeleteUser will delete the Elasticsearch user
func (cli *client) DeleteUser(user User) error {
	req, err := http.NewRequest("DELETE", fmt.Sprintf("/_security/user/%s", user.Username), nil)
	if err != nil {
		return err
	}

	response, err := cli.Perform(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != 200 && response.StatusCode != 404 {
		body, err := ioutil.ReadAll(response.Body)
		if err != nil {
			return err
		}
		return fmt.Errorf(string(body))
	}

	return nil
}

// UpdateUser will update the Elasticsearch users password and roles (if an roles are defined for the user). If the roles
// don't exist they will be created.
func (cli *client) UpdateUser(user User) error {
	var rolesToCreate []Role
	for _, role := range user.Roles {
		if role.Definition != nil {
			rolesToCreate = append(rolesToCreate, role)
		}
	}

	if len(rolesToCreate) > 0 {
		if err := cli.createRoles(rolesToCreate...); err != nil {
			return err
		}
	}

	j, err := json.Marshal(map[string]interface{}{
		"password": user.Password,
		"roles":    user.RoleNames(),
	})
	if err != nil {
		return err
	}

	req, err := http.NewRequest("PUT", fmt.Sprintf("/_security/user/%s", user.Username), bytes.NewBuffer(j))
	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", "application/json")

	response, err := cli.Perform(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		body, err := ioutil.ReadAll(response.Body)
		if err != nil {
			return err
		}
		return fmt.Errorf(string(body))
	}

	return nil
}

// UserExists queries Elasticsearch to see if a user with the given username already exists
func (cli *client) UserExists(username string) (bool, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("/_security/user/%s", username), nil)
	if err != nil {
		return false, err
	}

	response, err := cli.Perform(req)
	if err != nil {
		return false, err
	}
	response.Body.Close()

	return response.StatusCode == 200, nil
}

type esUsers map[string]esUser
type esUser struct {
	Roles    []string
}

// GetUsers returns all users stored in ES
func (cli *client) GetUsers() ([]User, error) {
	req, err := http.NewRequest("GET", "/_security/user", nil)
	if err != nil {
		return nil, err
	}

	response, err := cli.Perform(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	if response.StatusCode != 200 {
		return nil, fmt.Errorf(string(body))
	}

	var data esUsers
	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}
	var users []User
	for k, v := range data {
		var roles []Role
		for _, role := range v.Roles {
			roles = append(roles, Role{Name: role})
		}
		users = append(users, User{Username: k, Roles: roles})
	}

	if users == nil {
		return []User{}, nil
	}

	return users, nil
}
