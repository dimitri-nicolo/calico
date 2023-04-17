package fv

import (
	"fmt"

	"github.com/PaloAltoNetworks/pango/objs/addr"
	"github.com/PaloAltoNetworks/pango/objs/addrgrp"
	"github.com/PaloAltoNetworks/pango/objs/srvc"
	"github.com/PaloAltoNetworks/pango/poli/security"
	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"

	panutils "github.com/projectcalico/calico/firewall-integration/pkg/controllers/panorama/utils"
)

const InputDataFolder = "../../data/panorama/input/"
const ExpectedDataFolder = "../../data/panorama/expected/"

type MockPanoramaClientData struct {
	Addresses     []addr.Entry
	AddressGroups []addrgrp.Entry
	Services      []srvc.Entry
	Prerules      []security.Entry
	Postrules     []security.Entry
}

func getMockPanoramaClientData() (*MockPanoramaClientData, error) {
	clientData := &MockPanoramaClientData{}
	addrFileName := fmt.Sprintf("%s/%s", InputDataFolder, "addresses1.json")
	err := panutils.LoadData(addrFileName, &clientData.Addresses)
	if err != nil {
		return nil, err
	}

	addrgrpsFileName := fmt.Sprintf("%s/%s", InputDataFolder, "addressGroups1.json")
	err = panutils.LoadData(addrgrpsFileName, &clientData.AddressGroups)
	if err != nil {
		return nil, err
	}

	servicesFileName := fmt.Sprintf("%s/%s", InputDataFolder, "services1.json")
	err = panutils.LoadData(servicesFileName, &clientData.Services)
	if err != nil {
		return nil, err
	}

	preRulesFileName := fmt.Sprintf("%s/%s", InputDataFolder, "pre-rules1.json")
	err = panutils.LoadData(preRulesFileName, &clientData.Prerules)
	if err != nil {
		return nil, err
	}

	postRulesFileName := fmt.Sprintf("%s/%s", InputDataFolder, "post-rules1.json")
	err = panutils.LoadData(postRulesFileName, &clientData.Postrules)
	if err != nil {
		return nil, err
	}

	return clientData, nil
}

func getExpectedGnpMap(expectedFileName string) (map[string]v3.GlobalNetworkPolicy, error) {
	expectedGnpMap := map[string]v3.GlobalNetworkPolicy{}
	file := fmt.Sprintf("%s/%s.json", ExpectedDataFolder+"gnp/", expectedFileName)
	err := panutils.LoadData(file, &expectedGnpMap)
	if err != nil {
		return nil, err
	}

	return expectedGnpMap, nil
}

func getExpectedGnsMap(expectedFileName string) (map[string]v3.GlobalNetworkSet, error) {
	expectedGNSMap := map[string]v3.GlobalNetworkSet{}
	file := fmt.Sprintf("%s/%s.json", ExpectedDataFolder+"gns/", expectedFileName)
	err := panutils.LoadData(file, &expectedGNSMap)
	if err != nil {
		return nil, err
	}

	return expectedGNSMap, nil
}
