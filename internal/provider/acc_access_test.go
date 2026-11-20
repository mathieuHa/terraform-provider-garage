package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/mathieuHa/terraform-provider-garage/internal/client"
)

func TestAccKeyPermissionAlias(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc")
	importedID := "GK" + acctest.RandStringFromCharSet(24, "0123456789abcdef")
	importedSecret := acctest.RandStringFromCharSet(64, "0123456789abcdef")

	config := func(read, write, owner bool, expiration string) string {
		exp := ""
		if expiration != "" {
			exp = fmt.Sprintf("expiration = %q", expiration)
		}
		return fmt.Sprintf(`
resource "garage_bucket" "test" {
  global_alias = %[1]q
}

resource "garage_key" "app" {
  name                = "%[1]s-app"
  allow_create_bucket = true
  %[5]s
}

resource "garage_key" "imported" {
  id                = %[6]q
  secret_access_key = %[7]q
  name              = "%[1]s-imported"
}

resource "garage_bucket_permission" "app" {
  bucket_id     = garage_bucket.test.id
  access_key_id = garage_key.app.id
  read          = %[2]t
  write         = %[3]t
  owner         = %[4]t
}

resource "garage_bucket_permission" "imported" {
  bucket_id     = garage_bucket.test.id
  access_key_id = garage_key.imported.id
  read          = true
}

resource "garage_bucket_alias" "global" {
  bucket_id = garage_bucket.test.id
  alias     = "%[1]s-extra"
}

resource "garage_bucket_alias" "local" {
  bucket_id     = garage_bucket.test.id
  access_key_id = garage_key.app.id
  alias         = "mine"
}
`, name, read, write, owner, exp, importedID, importedSecret)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProviders,
		CheckDestroy:             checkBucketsDestroyed(name, name+"-extra"),
		Steps: []resource.TestStep{
			{
				Config: config(true, true, false, "2030-06-01T14:00:00+02:00"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("garage_key.app", "id", keyIDRegex),
					resource.TestMatchResourceAttr("garage_key.app", "secret_access_key", keySecretRegex),
					resource.TestCheckResourceAttr("garage_key.app", "allow_create_bucket", "true"),
					resource.TestCheckResourceAttr("garage_key.app", "expiration", "2030-06-01T14:00:00+02:00"),
					resource.TestCheckResourceAttr("garage_key.imported", "secret_access_key", importedSecret),
					resource.TestCheckResourceAttr("garage_bucket_permission.app", "owner", "false"),
					resource.TestCheckResourceAttr("garage_bucket_permission.imported", "write", "false"),
					checkBucket("garage_bucket.test", func(b *client.Bucket) error {
						if len(b.GlobalAliases) != 2 {
							return fmt.Errorf("global aliases = %v", b.GlobalAliases)
						}
						app := findKeyByName(b, name+"-app")
						if app == nil || !app.Permissions.Read || !app.Permissions.Write || app.Permissions.Owner {
							return fmt.Errorf("app permissions = %+v", app)
						}
						if len(app.BucketLocalAliases) != 1 || app.BucketLocalAliases[0] != "mine" {
							return fmt.Errorf("local aliases = %v", app.BucketLocalAliases)
						}
						return nil
					}),
				),
			},
			{
				Config: config(true, false, true, ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("garage_key.app", "expiration"),
					checkBucket("garage_bucket.test", func(b *client.Bucket) error {
						app := findKeyByName(b, name+"-app")
						if app == nil || !app.Permissions.Read || app.Permissions.Write || !app.Permissions.Owner {
							return fmt.Errorf("app permissions = %+v", app)
						}
						return nil
					}),
					func(s *terraform.State) error {
						k, err := testClient().GetKey(t0, s.RootModule().Resources["garage_key.app"].Primary.ID, false)
						if err != nil {
							return err
						}
						if k.Expiration != nil {
							return fmt.Errorf("expiration not cleared: %s", *k.Expiration)
						}
						return nil
					},
				),
			},
			{ResourceName: "garage_key.app", ImportState: true, ImportStateVerify: true},
			{ResourceName: "garage_key.imported", ImportState: true, ImportStateVerify: true},
			{ResourceName: "garage_bucket_permission.app", ImportState: true, ImportStateVerify: true},
			{ResourceName: "garage_bucket_alias.global", ImportState: true, ImportStateVerify: true},
			{ResourceName: "garage_bucket_alias.local", ImportState: true, ImportStateVerify: true},
			{
				PreConfig: func() {
					b, err := testClient().GetBucketByGlobalAlias(t0, name)
					if err != nil {
						t.Fatal(err)
					}
					app := findKeyByName(b, name+"-app")
					if _, err := testClient().DenyBucketKey(t0, b.ID, app.AccessKeyID, client.BucketPermissions{Read: true}); err != nil {
						t.Fatal(err)
					}
					if _, err := testClient().RemoveBucketAlias(t0, b.ID, "mine", app.AccessKeyID); err != nil {
						t.Fatal(err)
					}
				},
				Config:             config(true, false, true, ""),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
			{
				Config: config(true, false, true, ""),
				Check: checkBucket("garage_bucket.test", func(b *client.Bucket) error {
					app := findKeyByName(b, name+"-app")
					if app == nil || !app.Permissions.Read || len(app.BucketLocalAliases) != 1 {
						return fmt.Errorf("drift not corrected: %+v", app)
					}
					return nil
				}),
			},
		},
	})
}

func findKeyByName(b *client.Bucket, name string) *client.BucketKey {
	for i := range b.Keys {
		if b.Keys[i].Name == name {
			return &b.Keys[i]
		}
	}
	return nil
}
