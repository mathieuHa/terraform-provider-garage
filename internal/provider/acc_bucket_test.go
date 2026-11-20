package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/mathieuHa/terraform-provider-garage/internal/client"
)

func TestAccBucket(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc")
	renamed := name + "-renamed"
	addr := "garage_bucket.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProviders,
		CheckDestroy:             checkBucketsDestroyed(name, renamed),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "garage_bucket" "test" {
  global_alias = %q
  max_size     = 1073741824
}`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr(addr, "id", bucketIDRegex),
					resource.TestCheckResourceAttr(addr, "max_size", "1073741824"),
					resource.TestCheckNoResourceAttr(addr, "max_objects"),
					resource.TestCheckNoResourceAttr(addr, "website"),
					checkBucket(addr, func(b *client.Bucket) error {
						if b.Quotas.MaxSize == nil || *b.Quotas.MaxSize != 1073741824 {
							return fmt.Errorf("quota not set: %+v", b.Quotas)
						}
						return nil
					}),
				),
			},
			{
				Config: fmt.Sprintf(`
resource "garage_bucket" "test" {
  global_alias = %q
  max_objects  = 1000

  website = {
    index_document = "index.html"
    error_document = "404.html"
    routing_rules = [
      {
        condition = { key_prefix_equals = "old/", http_error_code_returned_equals = 404 }
        redirect  = { replace_key_prefix_with = "new/", http_redirect_code = 301, protocol = "https", host_name = "example.com" }
      },
      {
        redirect = { replace_key_with = "index.html" }
      },
    ]
  }

  cors_rules = [
    {
      id              = "app"
      allowed_origins = ["https://app.example.com"]
      allowed_methods = ["GET", "PUT"]
      allowed_headers = ["*"]
      expose_headers  = ["ETag"]
      max_age_seconds = 3600
    },
    {
      allowed_origins = ["*"]
      allowed_methods = ["GET"]
    },
  ]

  lifecycle_rules = [
    {
      id              = "tmp"
      filter          = { prefix = "tmp/", object_size_greater_than = 10 }
      expiration_days = 7
    },
    {
      id                                     = "uploads"
      enabled                                = false
      abort_incomplete_multipart_upload_days = 2
    },
    {
      filter          = { prefix = "archive/" }
      expiration_date = "2030-01-01T00:00:00Z"
    },
  ]
}`, renamed),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(addr, "global_alias", renamed),
					resource.TestCheckNoResourceAttr(addr, "max_size"),
					resource.TestCheckResourceAttr(addr, "max_objects", "1000"),
					resource.TestCheckResourceAttr(addr, "website.routing_rules.1.redirect.http_redirect_code", "302"),
					resource.TestCheckResourceAttr(addr, "cors_rules.#", "2"),
					resource.TestCheckNoResourceAttr(addr, "cors_rules.1.allowed_headers"),
					resource.TestCheckResourceAttr(addr, "lifecycle_rules.0.enabled", "true"),
					resource.TestCheckResourceAttr(addr, "lifecycle_rules.1.enabled", "false"),
					checkBucket(addr, func(b *client.Bucket) error {
						if len(b.GlobalAliases) != 1 || b.GlobalAliases[0] != renamed {
							return fmt.Errorf("aliases = %v", b.GlobalAliases)
						}
						if !b.WebsiteAccess || b.WebsiteConfig == nil || len(b.WebsiteConfig.RoutingRules) != 2 {
							return fmt.Errorf("website = %+v", b.WebsiteConfig)
						}
						if len(b.CorsRules) != 2 || len(b.LifecycleRules) != 3 {
							return fmt.Errorf("cors = %d rules, lifecycle = %d rules", len(b.CorsRules), len(b.LifecycleRules))
						}
						if f := b.LifecycleRules[0].Filter; f == nil || f.And == nil {
							return fmt.Errorf("a two-condition filter must be wrapped in And: %+v", f)
						}
						return nil
					}),
				),
			},
			{
				ResourceName:      addr,
				ImportState:       true,
				ImportStateId:     renamed,
				ImportStateVerify: true,
			},
			{
				Config: fmt.Sprintf(`
resource "garage_bucket" "test" {
  global_alias    = %q
  cors_rules      = []
  lifecycle_rules = []
}`, renamed),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr(addr, "website"),
					resource.TestCheckNoResourceAttr(addr, "max_objects"),
					resource.TestCheckResourceAttr(addr, "cors_rules.#", "0"),
					checkBucket(addr, func(b *client.Bucket) error {
						if b.WebsiteAccess || len(b.CorsRules) != 0 || len(b.LifecycleRules) != 0 || b.Quotas.MaxObjects != nil {
							return fmt.Errorf("bucket not cleared: website=%v cors=%d lifecycle=%d quotas=%+v", b.WebsiteAccess, len(b.CorsRules), len(b.LifecycleRules), b.Quotas)
						}
						return nil
					}),
				),
			},
		},
	})
}

func TestAccBucket_unmanagedRulesAndDrift(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc")
	addr := "garage_bucket.test"
	config := fmt.Sprintf(`
resource "garage_bucket" "test" {
  global_alias = %q
}`, name)
	managed := fmt.Sprintf(`
resource "garage_bucket" "test" {
  global_alias    = %q
  lifecycle_rules = [{ expiration_days = 30 }]
}`, name)
	var id string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProviders,
		CheckDestroy:             checkBucketsDestroyed(name),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: func(s *terraform.State) error {
					id = s.RootModule().Resources[addr].Primary.ID
					return nil
				},
			},
			{
				PreConfig: func() {
					days := int64(1)
					rules := []client.LifecycleRule{{Status: "Enabled", Expiration: &client.LifecycleExpiration{Days: &days}}}
					cors := []client.CorsRule{{AllowedOrigin: []string{"*"}, AllowedMethod: []string{"GET"}}}
					if _, err := testClient().UpdateBucket(t0, id, client.UpdateBucketRequest{LifecycleRules: &rules, CorsRules: &cors}); err != nil {
						t.Fatal(err)
					}
				},
				Config:   config,
				PlanOnly: true,
			},
			{
				Config: managed,
				Check: checkBucket(addr, func(b *client.Bucket) error {
					if len(b.LifecycleRules) != 1 || *b.LifecycleRules[0].Expiration.Days != 30 {
						return fmt.Errorf("lifecycle = %+v", b.LifecycleRules)
					}
					if len(b.CorsRules) != 1 {
						return fmt.Errorf("unmanaged CORS rules were changed: %+v", b.CorsRules)
					}
					return nil
				}),
			},
			{
				PreConfig: func() {
					rules := []client.LifecycleRule{}
					if _, err := testClient().UpdateBucket(t0, id, client.UpdateBucketRequest{LifecycleRules: &rules, Quotas: &client.Quotas{MaxObjects: ptr(int64(5))}}); err != nil {
						t.Fatal(err)
					}
				},
				Config:             managed,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
			{
				Config: managed,
				Check: checkBucket(addr, func(b *client.Bucket) error {
					if len(b.LifecycleRules) != 1 || b.Quotas.MaxObjects != nil {
						return fmt.Errorf("drift not corrected: lifecycle=%+v quotas=%+v", b.LifecycleRules, b.Quotas)
					}
					return nil
				}),
			},
		},
	})
}

func ptr[T any](v T) *T { return &v }

func checkBucket(addr string, check func(*client.Bucket) error) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[addr]
		if !ok {
			return fmt.Errorf("%s not in state", addr)
		}
		b, err := testClient().GetBucket(t0, rs.Primary.ID)
		if err != nil {
			return err
		}
		return check(b)
	}
}

func checkBucketsDestroyed(aliases ...string) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		for _, alias := range aliases {
			if _, err := testClient().GetBucketByGlobalAlias(t0, alias); !client.IsNotFound(err) {
				return fmt.Errorf("bucket %s still exists: %w", alias, err)
			}
		}
		return nil
	}
}
