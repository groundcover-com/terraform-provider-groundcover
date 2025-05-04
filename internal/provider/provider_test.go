// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

// testAccProtoV6ProviderFactories is used to instantiate a provider during acceptance testing.
// The factory function is called for each Terraform CLI command to create a provider
// server that the CLI can connect to and interact with.
/* // Commented out as no acceptance tests currently use this
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"groundcover": providerserver.NewProtocol6WithError(New("test")()),
}
*/

// testAccProtoV6ProviderFactoriesWithEcho includes the echo provider alongside the groundcover provider.
// It allows for testing assertions on data returned by an ephemeral resource during Open.
// The echoprovider is used to arrange tests by echoing ephemeral data into the Terraform state.
// This lets the data be referenced in test assertions with state checks.
/* // Commented out as no acceptance tests currently use this
var testAccProtoV6ProviderFactoriesWithEcho = map[string]func() (tfprotov6.ProviderServer, error){
	"groundcover": providerserver.NewProtocol6WithError(New("test")()),
	"echo":        echoprovider.NewProviderServer(),
}
*/

/* // Commented out as no acceptance tests currently use this
func testAccPreCheck(t *testing.T) {
	// You can add code here to run prior to any test case execution, for example assertions
	// about the appropriate environment variables being set are common to see in a pre-check
	// function.
}
*/
