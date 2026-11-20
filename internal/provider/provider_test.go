package provider

import (
	"context"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/mathieuHa/terraform-provider-garage/internal/client"
)

var t0 = context.Background()

var testAccProviders = map[string]func() (tfprotov6.ProviderServer, error){
	"garage": providerserver.NewProtocol6WithError(New("test")()),
}

func testAccPreCheck(t *testing.T) {
	t.Helper()
	for _, env := range []string{"GARAGE_ENDPOINT", "GARAGE_TOKEN"} {
		if os.Getenv(env) == "" {
			t.Fatalf("%s must be set for acceptance tests (see scripts/testacc-garage.sh)", env)
		}
	}
}

func testClient() *client.Client {
	return client.New(os.Getenv("GARAGE_ENDPOINT"), os.Getenv("GARAGE_TOKEN"), "acctest", nil)
}
