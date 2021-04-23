// Copyright (c) 2020 Tigera, Inc. All rights reserved.

package fv

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/projectcalico/apiserver/pkg/authentication"
)

func NewAuthnClient() authentication.Authenticator {
	authenticator := authentication.NewFakeAuthenticator()
	basic := map[string][]string{}
	bfile, err := os.Open("./basic_auth.csv")
	if err != nil {
		log.Fatal(err)
	}
	defer bfile.Close()

	// Header of the CSV in this test: password,user,id,group1,group2,...
	scanner := bufio.NewScanner(bfile)
	for scanner.Scan() {
		var csv = strings.Split(scanner.Text(), ",")
		concat := fmt.Sprintf("%s:%s", csv[1], csv[0])
		b64 := base64.StdEncoding.EncodeToString([]byte(concat))
		basic[fmt.Sprintf("Basic %s", b64)] = csv
		if len(csv) == 4 {
			authenticator.AddValidApiResponse(fmt.Sprintf("Basic %s", b64), csv[1], []string{csv[3], "system:authenticated"})
		} else {
			authenticator.AddValidApiResponse(fmt.Sprintf("Basic %s", b64), csv[1], []string{"system:authenticated"})
		}
	}

	token := map[string][]string{}
	tfile, err := os.Open("./token_auth.csv")
	if err != nil {
		log.Fatal(err)
	}
	defer tfile.Close()

	// Header of the CSV in this test: password,user,id,group1,group2,...
	scanner = bufio.NewScanner(tfile)
	for scanner.Scan() {
		csv := strings.Split(scanner.Text(), ",")
		token[fmt.Sprintf("Bearer %s", csv[0])] = csv
		authenticator.AddValidApiResponse(fmt.Sprintf("Bearer %s", csv[0]), csv[1], []string{"system:authenticated"})
	}

	// Add some error test-cases to the authenticator
	authenticator.AddErrorAPIServerResponse("Bearer d00dbeef", nil, http.StatusUnauthorized)
	// The line below is for user basicuserall:badpw
	authenticator.AddErrorAPIServerResponse("Basic YmFzaWN1c2VyYWxsOmJhZHB3", []byte{}, http.StatusUnauthorized)

	return authenticator
}
