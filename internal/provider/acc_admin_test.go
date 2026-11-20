package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/mathieuHa/terraform-provider-garage/internal/client"
)

func TestAccAdminToken(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc")
	addr := "garage_admin_token.test"
	config := func(scope, expiration string) string {
		exp := ""
		if expiration != "" {
			exp = fmt.Sprintf("expiration = %q", expiration)
		}
		return fmt.Sprintf(`
resource "garage_admin_token" "test" {
  name  = %q
  scope = %s
  %s
}`, name, scope, exp)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProviders,
		CheckDestroy: func(s *terraform.State) error {
			tokens, err := testClient().ListAdminTokens(t0)
			if err != nil {
				return err
			}
			for _, tok := range tokens {
				if tok.Name == name {
					return fmt.Errorf("admin token %s still exists", name)
				}
			}
			return nil
		},
		Steps: []resource.TestStep{
			{
				Config: config(`["GetClusterHealth", "ListBuckets"]`, "2030-01-01T00:00:00Z"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(addr, "secret_token"),
					resource.TestCheckResourceAttr(addr, "scope.#", "2"),
					func(s *terraform.State) error {
						secret := s.RootModule().Resources[addr].Primary.Attributes["secret_token"]
						scoped := client.New(os.Getenv("GARAGE_ENDPOINT"), secret, "acctest", nil)
						if _, err := scoped.ListBuckets(t0); err != nil {
							return fmt.Errorf("scoped token cannot call an allowed endpoint: %w", err)
						}
						if _, err := scoped.ListKeys(t0); err == nil {
							return fmt.Errorf("scoped token can call an endpoint outside its scope")
						}
						return nil
					},
				),
			},
			{
				Config: config(`["GetClusterHealth", "ListBuckets", "ListKeys"]`, ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(addr, "scope.#", "3"),
					resource.TestCheckNoResourceAttr(addr, "expiration"),
					func(s *terraform.State) error {
						secret := s.RootModule().Resources[addr].Primary.Attributes["secret_token"]
						_, err := client.New(os.Getenv("GARAGE_ENDPOINT"), secret, "acctest", nil).ListKeys(t0)
						return err
					},
				),
			},
			{
				ResourceName:            addr,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"secret_token"},
			},
		},
	})
}

func TestAccClusterLayout(t *testing.T) {
	node := os.Getenv("GARAGE_TEST_NODE_ID")
	if node == "" {
		t.Skip("GARAGE_TEST_NODE_ID is not set")
	}
	addr := "garage_cluster_layout.test"
	config := func(capacity int64, tags string) string {
		return fmt.Sprintf(`
resource "garage_cluster_layout" "test" {
  roles = {
    %q = {
      zone     = "dc1"
      capacity = %d
      tags     = %s
    }
  }
  zone_redundancy = 1
}`, node, capacity, tags)
	}
	var version int64

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: config(2000000000, `["ssd", "rack-a"]`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(addr, fmt.Sprintf("roles.%s.capacity", node), "2000000000"),
					func(_ *terraform.State) error {
						layout, err := testClient().GetClusterLayout(t0)
						if err != nil {
							return err
						}
						version = layout.Version
						if len(layout.Roles) != 1 || *layout.Roles[0].Capacity != 2000000000 || len(layout.Roles[0].Tags) != 2 {
							return fmt.Errorf("layout = %+v", layout.Roles)
						}
						if layout.Parameters.ZoneRedundancy.AtLeast == nil || *layout.Parameters.ZoneRedundancy.AtLeast != 1 {
							return fmt.Errorf("zone redundancy not applied")
						}
						return nil
					},
				),
			},
			{
				Config:   config(2000000000, `["rack-a", "ssd"]`),
				PlanOnly: true,
			},
			{
				Config: config(3000000000, `["ssd"]`),
				Check: func(_ *terraform.State) error {
					layout, err := testClient().GetClusterLayout(t0)
					if err != nil {
						return err
					}
					if layout.Version != version+1 || *layout.Roles[0].Capacity != 3000000000 {
						return fmt.Errorf("expected version %d with the new capacity, got %d %+v", version+1, layout.Version, layout.Roles)
					}
					return nil
				},
			},
			{
				ResourceName:      addr,
				ImportState:       true,
				ImportStateId:     "cluster_layout",
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccDataSources(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "garage_bucket" "test" {
  global_alias = %[1]q
  max_objects  = 10
}

resource "garage_key" "test" {
  name = "%[1]s-key"
}

resource "garage_bucket_permission" "test" {
  bucket_id     = garage_bucket.test.id
  access_key_id = garage_key.test.id
  read          = true
}

data "garage_bucket" "by_alias" {
  global_alias = garage_bucket.test.global_alias
  depends_on   = [garage_bucket_permission.test]
}

data "garage_key" "by_name" {
  name       = garage_key.test.name
  depends_on = [garage_bucket_permission.test]
}

data "garage_buckets" "all" {
  depends_on = [garage_bucket.test]
}

data "garage_keys" "all" {
  depends_on = [garage_key.test]
}

data "garage_admin_token" "current" {}
data "garage_cluster_health" "this" {}
data "garage_cluster_status" "this" {}
data "garage_cluster_layout" "this" {}
data "garage_node" "self" {}
`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.garage_bucket.by_alias", "id", "garage_bucket.test", "id"),
					resource.TestCheckResourceAttr("data.garage_bucket.by_alias", "max_objects", "10"),
					resource.TestCheckResourceAttr("data.garage_bucket.by_alias", "keys.0.read", "true"),
					resource.TestCheckResourceAttrPair("data.garage_key.by_name", "id", "garage_key.test", "id"),
					resource.TestCheckResourceAttr("data.garage_key.by_name", "buckets.#", "1"),
					resource.TestCheckResourceAttrSet("data.garage_buckets.all", "buckets.0.id"),
					resource.TestCheckResourceAttrSet("data.garage_keys.all", "keys.0.id"),
					resource.TestCheckResourceAttr("data.garage_admin_token.current", "scope.0", "*"),
					resource.TestCheckResourceAttr("data.garage_cluster_health.this", "status", "healthy"),
					resource.TestCheckResourceAttr("data.garage_cluster_status.this", "nodes.#", "1"),
					resource.TestCheckResourceAttr("data.garage_cluster_layout.this", "roles.#", "1"),
					resource.TestCheckResourceAttrSet("data.garage_node.self", "garage_version"),
				),
			},
		},
	})
}
