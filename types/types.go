package types

type License struct {
	Id     string   `json:"id"`
	Key    string   `json:"key" validate:"required"`
	Name   string   `json:"name" validate:"required"`
	Claims []string `json:"claims"`
	Jwt    string   `json:"jwt"`
}
