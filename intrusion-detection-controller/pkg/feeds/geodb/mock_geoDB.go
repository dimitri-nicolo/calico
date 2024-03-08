package geodb

import (
	"net"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
)

type MockGeoDB struct {
}

func (mg *MockGeoDB) City(ip net.IP) (v1.IPGeoInfo, error) {
	return v1.IPGeoInfo{
		CityName:    "Naucelles",
		CountryName: "France",
		ISO:         "FR",
		ASN:         "",
	}, nil
}

func (mg *MockGeoDB) ASN(ip net.IP) (string, error) {
	return "3215", nil
}
