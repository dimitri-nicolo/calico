// Copyright (c) 2020-2021 Tigera, Inc. All rights reserved.
package metrics_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	apiv3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	"github.com/tigera/license-agent/pkg/metrics"
	"time"
)

const validLicenseCertificate = `-----BEGIN CERTIFICATE-----
MIIFxjCCA66gAwIBAgIQVq3rz5D4nQF1fIgMEh71DzANBgkqhkiG9w0BAQsFADCB
tTELMAkGA1UEBhMCVVMxEzARBgNVBAgTCkNhbGlmb3JuaWExFjAUBgNVBAcTDVNh
biBGcmFuY2lzY28xFDASBgNVBAoTC1RpZ2VyYSwgSW5jMSIwIAYDVQQLDBlTZWN1
cml0eSA8c2lydEB0aWdlcmEuaW8+MT8wPQYDVQQDEzZUaWdlcmEgRW50aXRsZW1l
bnRzIEludGVybWVkaWF0ZSBDZXJ0aWZpY2F0ZSBBdXRob3JpdHkwHhcNMTgwNDA1
MjEzMDI5WhcNMjAxMDA2MjEzMDI5WjCBnjELMAkGA1UEBhMCVVMxEzARBgNVBAgT
CkNhbGlmb3JuaWExFjAUBgNVBAcTDVNhbiBGcmFuY2lzY28xFDASBgNVBAoTC1Rp
Z2VyYSwgSW5jMSIwIAYDVQQLDBlTZWN1cml0eSA8c2lydEB0aWdlcmEuaW8+MSgw
JgYDVQQDEx9UaWdlcmEgRW50aXRsZW1lbnRzIENlcnRpZmljYXRlMIIBojANBgkq
hkiG9w0BAQEFAAOCAY8AMIIBigKCAYEAwg3LkeHTwMi651af/HEXi1tpM4K0LVqb
5oUxX5b5jjgi+LHMPzMI6oU+NoGPHNqirhAQqK/k7W7r0oaMe1APWzaCAZpHiMxE
MlsAXmLVUrKg/g+hgrqeije3JDQutnN9h5oZnsg1IneBArnE/AKIHH8XE79yMG49
LaKpPGhpF8NoG2yoWFp2ekihSohvqKxa3m6pxoBVdwNxN0AfWxb60p2SF0lOi6B3
hgK6+ILy08ZqXefiUs+GC1Af4qI1jRhPkjv3qv+H1aQVrq6BqKFXwWIlXCXF57CR
hvUaTOG3fGtlVyiPE4+wi7QDo0cU/+Gx4mNzvmc6lRjz1c5yKxdYvgwXajSBx2pw
kTP0iJxI64zv7u3BZEEII6ak9mgUU1CeGZ1KR2Xu80JiWHAYNOiUKCBYHNKDCUYl
RBErYcAWz2mBpkKyP6hbH16GjXHTTdq5xENmRDHabpHw5o+21LkWBY25EaxjwcZa
Y3qMIOllTZ2iRrXu7fSP6iDjtFCcE2bFAgMBAAGjZzBlMA4GA1UdDwEB/wQEAwIF
oDATBgNVHSUEDDAKBggrBgEFBQcDAjAdBgNVHQ4EFgQUIY7LzqNTzgyTBE5efHb5
kZ71BUEwHwYDVR0jBBgwFoAUxZA5kifzo4NniQfGKb+4wruTIFowDQYJKoZIhvcN
AQELBQADggIBAAK207LaqMrnphF6CFQnkMLbskSpDZsKfqqNB52poRvUrNVUOB1w
3dSEaBUjhFgUU6yzF+xnuH84XVbjD7qlM3YbdiKvJS9jrm71saCKMNc+b9HSeQAU
DGY7GPb7Y/LG0GKYawYJcPpvRCNnDLsSVn5N4J1foWAWnxuQ6k57ymWwcddibYHD
OPakOvO4beAnvax3+K5dqF0bh2Np79YolKdIgUVzf4KSBRN4ZE3AOKlBfiKUvWy6
nRGvu8O/8VaI0vGaOdXvWA5b61H0o5cm50A88tTm2LHxTXynE3AYriHxsWBbRpoM
oFnmDaQtGY67S6xGfQbwxrwCFd1l7rGsyBQ17cuusOvMNZEEWraLY/738yWKw3qX
U7KBxdPWPIPd6iDzVjcZrS8AehUEfNQ5yd26gDgW+rZYJoAFYv0vydMEyoI53xXs
cpY84qV37ZC8wYicugidg9cFtD+1E0nVgOLXPkHnmc7lIDHFiWQKfOieH+KoVCbb
zdFu3rhW31ygphRmgszkHwApllCTBBMOqMaBpS8eHCnetOITvyB4Kiu1/nKvVxhY
exit11KQv8F3kTIUQRm0qw00TSBjuQHKoG83yfimlQ8OazciT+aLpVaY8SOrrNnL
IJ8dHgTpF9WWHxx04DDzqrT7Xq99F9RzDzM7dSizGxIxonoWcBjiF6n5
-----END CERTIFICATE-----`

const validLicenseToken = `eyJhbGciOiJBMTI4R0NNS1ciLCJjdHkiOiJKV1QiLCJlbmMiOiJBMTI4R0NNIiwiaXYiOiJJOTdyUWRaMVNXaUtSSXdGIiwidGFnIjoiZVFmTnNhcjBvcHdJTXJBRnREQXpvUSIsInR5cCI6IkpXVCJ9.S87Ufx8wtm-y3JmIHN6D7g.JJ3h6_KAQ7ntwwOH.ddzfQY2M_wWqZFoh8lr4pMqs8bz777Eh_5JFAvlYcu7cN35EAYhHWiPL6FUSiZz-pVRSdfNZ1vGwkWSnn7r4002YO-ojdnr2Ua1JJkFoGzbxKNK-iGP-I3fHKvnvYB7EC6KxtsK-4vS7fsoYFXm8puqedRXzvoSKFgjQZKMEWEI5t_GJ7dhY1llWYQyKozWnWnNEa_teZDSXfXaMh8k7AEh5zkRs8I7KW0nocPWT2yV8AjkHNdQjBF--Id16GifuYrwpuCgus22oWZ-dCxU9DOuSep_brKZHqtR6lU30c7uuNJdquqbepHssAgjLxIqb0eKSsTtYR-jaA6wvaMDdOSQSWH0XsIPjhfuZ7yQm-hi-Fv41QgmmXIntJ69DIhMU54WsmLz0Oee2EIad9ThJOBj-10Nl4y8wVAAweTiazCmYhza0a_-XDZScgEpdoAIIqMGrt_bzo6uoRPiAfIaxZjBTepbK0jXtDDTvf-zsFbRsclWbm1rLu0tXe-YLY3iVgox1N_NCv36JNcfud1I2Lpue70cbkpbu0RG8Vv0jcZncFU-nYyXsSK0ol6bPIDSHE3BUmZJpUeT-kPS7hQgSXMapvd5UrMLTbWhR-OgDIcqo7UUVm6O48ZE3T3P-5Qu6f-VHbm5AXHpVfew6Zit5-Voi14nJz7HmcQa8b3WTKmyggnVxt47VlBrzdVj_bJJLdBBj2i-l2_3tXdtj4nUEBACC8UBpR1GilDy4WHnhmjwRZtws_N8j-gLjXgTCxnVS6B_9ImVhZT-HOKkmBtXAGsSy7GySG6MdWzkfiuQHRZU4ferUKK-VFWSxFNrIcnbErWyPdW10efg74mfyxpVY60k2BeB1HxHAEKhcT8woaJQSjEIkIe246fA6D7P_p4BbZR-rNm0KGZa6UChtyTe2-v6tNvplsAYV-twmnyELPE3iSQdNAtigJ5Z4GqBKXr_cFiqQTGpSS49bqsxMlY2qG1HXZFn2mvtPsJm4UuF-RaL3i6LdkKaG8yM0tm1TAFldpepy7icaW9tbgJ9-LeM8kTa-rvboifsAA98ot-TdhnsI44vxD6mNlDsUfJHcfO5wNLKbBlrZ7BAf5ox46BBY40TRWIyaefk_HoIthZiMKEA.U2ayMukyzhFXcY24bqBGKA`

var licenseKey = struct {
	description string
	license     apiv3.LicenseKey
}{
	description: "Test License",
	license: apiv3.LicenseKey{
		Spec: apiv3.LicenseKeySpec{
			Token:       validLicenseToken,
			Certificate: validLicenseCertificate,
		},
	},
}

var _ = Describe("Check License Validity", func() {
	maxLicensedNodes := 100
	It("Validates test license", func() {
		By("Checking timeDuration")
		min, err := time.ParseDuration("1m")
		Expect(err).ShouldNot(HaveOccurred())

		By("Checking Validity")
		lr := metrics.NewLicenseReporter("", "", "", "", min, 9081)
		isValid, _, maxNodes := lr.LicenseHandler(licenseKey.license)
		Expect(isValid).To(BeTrue(), "License Valid")
		Expect(maxNodes).Should(Equal(maxLicensedNodes))

	})
})
