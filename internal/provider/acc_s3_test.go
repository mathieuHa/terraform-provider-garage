package provider

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func testAccS3PreCheck(t *testing.T) {
	t.Helper()
	testAccPreCheck(t)
	for _, env := range []string{"GARAGE_TEST_S3_ENDPOINT", "GARAGE_TEST_WEB_ENDPOINT"} {
		if os.Getenv(env) == "" {
			t.Fatalf("%s must be set for S3 tests (see scripts/testacc-garage.sh)", env)
		}
	}
}

func s3For(s *terraform.State, keyAddr string) (*s3.Client, error) {
	rs, ok := s.RootModule().Resources[keyAddr]
	if !ok {
		return nil, fmt.Errorf("%s not in state", keyAddr)
	}
	return s3.New(s3.Options{
		BaseEndpoint:               aws.String(os.Getenv("GARAGE_TEST_S3_ENDPOINT")),
		Region:                     "garage",
		UsePathStyle:               true,
		Credentials:                credentials.NewStaticCredentialsProvider(rs.Primary.Attributes["id"], rs.Primary.Attributes["secret_access_key"], ""),
		RequestChecksumCalculation: aws.RequestChecksumCalculationWhenRequired,
		ResponseChecksumValidation: aws.ResponseChecksumValidationWhenRequired,
	}), nil
}

func s3Put(c *s3.Client, bucket, key, body string) error {
	_, err := c.PutObject(t0, &s3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(key),
		Body:        strings.NewReader(body),
		ContentType: aws.String("text/html"),
	})
	return err
}

func s3Get(c *s3.Client, bucket, key string) (string, error) {
	out, err := c.GetObject(t0, &s3.GetObjectInput{Bucket: aws.String(bucket), Key: aws.String(key)})
	if err != nil {
		return "", err
	}
	defer func() { _ = out.Body.Close() }()
	data, err := io.ReadAll(out.Body)
	return string(data), err
}

func s3Delete(c *s3.Client, bucket, key string) error {
	_, err := c.DeleteObject(t0, &s3.DeleteObjectInput{Bucket: aws.String(bucket), Key: aws.String(key)})
	return err
}

func apiErrorCode(err error) string {
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		return apiErr.ErrorCode()
	}
	return ""
}

func expectCode(what string, err error, code string) error {
	if err == nil {
		return fmt.Errorf("%s: succeeded, expected %s", what, code)
	}
	if got := apiErrorCode(err); got != code {
		return fmt.Errorf("%s: expected %s, got %w", what, code, err)
	}
	return nil
}

func expectOK(what string, err error) error {
	if err != nil {
		return fmt.Errorf("%s: %w", what, err)
	}
	return nil
}

type s3Check func(s *terraform.State) error

func s3Steps(checks ...s3Check) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, check := range checks {
			if err := check(s); err != nil {
				return err
			}
		}
		return nil
	}
}

func TestAccS3_permissions(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-s3")
	created := name + "-by-key"
	config := func(readerWrite, readerGranted bool) string {
		reader := ""
		if readerGranted {
			reader = fmt.Sprintf(`
resource "garage_bucket_permission" "reader" {
  bucket_id     = garage_bucket.test.id
  access_key_id = garage_key.reader.id
  read          = true
  write         = %t
}`, readerWrite)
		}
		return fmt.Sprintf(`
resource "garage_bucket" "test" {
  global_alias = %[1]q
}

resource "garage_key" "writer" {
  name = "%[1]s-writer"
}

resource "garage_key" "reader" {
  name = "%[1]s-reader"
}

resource "garage_key" "outsider" {
  name = "%[1]s-outsider"
}

resource "garage_key" "creator" {
  name                = "%[1]s-creator"
  allow_create_bucket = true
}

resource "garage_bucket_permission" "writer" {
  bucket_id     = garage_bucket.test.id
  access_key_id = garage_key.writer.id
  read          = true
  write         = true
}

resource "garage_bucket_alias" "writer_local" {
  bucket_id     = garage_bucket.test.id
  access_key_id = garage_key.writer.id
  alias         = "mine"
}
%[2]s
`, name, reader)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccS3PreCheck(t) },
		ProtoV6ProviderFactories: testAccProviders,
		CheckDestroy:             checkBucketsDestroyed(name, created),
		Steps: []resource.TestStep{
			{
				Config: config(false, true),
				Check: s3Steps(
					func(s *terraform.State) error {
						writer, _ := s3For(s, "garage_key.writer")
						reader, _ := s3For(s, "garage_key.reader")
						outsider, _ := s3For(s, "garage_key.outsider")
						if err := expectOK("writer puts an object", s3Put(writer, name, "a.txt", "hello")); err != nil {
							return err
						}
						if body, err := s3Get(reader, name, "a.txt"); err != nil || body != "hello" {
							return fmt.Errorf("reader reads the object: %q, %w", body, err)
						}
						if err := expectCode("reader puts an object", s3Put(reader, name, "b.txt", "x"), "AccessDenied"); err != nil {
							return err
						}
						if err := expectCode("reader deletes an object", s3Delete(reader, name, "a.txt"), "AccessDenied"); err != nil {
							return err
						}
						_, err := s3Get(outsider, name, "a.txt")
						return expectCode("a key without permission reads the object", err, "AccessDenied")
					},
					func(s *terraform.State) error {
						writer, _ := s3For(s, "garage_key.writer")
						reader, _ := s3For(s, "garage_key.reader")
						if body, err := s3Get(writer, "mine", "a.txt"); err != nil || body != "hello" {
							return fmt.Errorf("writer reads through its local alias: %q, %w", body, err)
						}
						_, err := s3Get(reader, "mine", "a.txt")
						return expectCode("another key uses the local alias", err, "NoSuchBucket")
					},
					func(s *terraform.State) error {
						writer, _ := s3For(s, "garage_key.writer")
						creator, _ := s3For(s, "garage_key.creator")
						_, err := writer.CreateBucket(t0, &s3.CreateBucketInput{Bucket: aws.String(created)})
						if err := expectCode("a key without allow_create_bucket creates a bucket", err, "AccessDenied"); err != nil {
							return err
						}
						if _, err := creator.CreateBucket(t0, &s3.CreateBucketInput{Bucket: aws.String(created)}); err != nil {
							return fmt.Errorf("a key with allow_create_bucket creates a bucket: %w", err)
						}
						_, err = creator.DeleteBucket(t0, &s3.DeleteBucketInput{Bucket: aws.String(created)})
						return expectOK("the creating key deletes its bucket", err)
					},
				),
			},
			{
				Config: config(true, true),
				Check: func(s *terraform.State) error {
					reader, _ := s3For(s, "garage_key.reader")
					writer, _ := s3For(s, "garage_key.writer")
					if err := expectOK("reader puts an object once write is granted", s3Put(reader, name, "b.txt", "x")); err != nil {
						return err
					}
					if body, err := s3Get(writer, name, "b.txt"); err != nil || body != "x" {
						return fmt.Errorf("writer reads the object written by reader: %q, %w", body, err)
					}
					return expectOK("reader deletes its object", s3Delete(reader, name, "b.txt"))
				},
			},
			{
				Config: config(true, false),
				Check: func(s *terraform.State) error {
					writer, _ := s3For(s, "garage_key.writer")
					reader, _ := s3For(s, "garage_key.reader")
					_, err := s3Get(reader, name, "a.txt")
					if err := expectCode("reader reads after its permission is destroyed", err, "AccessDenied"); err != nil {
						return err
					}
					return expectOK("writer cleans up", s3Delete(writer, name, "a.txt"))
				},
			},
		},
	})
}

func TestAccS3_quota(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-s3")
	config := func(maxObjects int) string {
		return fmt.Sprintf(`
resource "garage_bucket" "test" {
  global_alias = %[1]q
  max_objects  = %[2]d
}

resource "garage_key" "writer" {
  name = "%[1]s-writer"
}

resource "garage_bucket_permission" "writer" {
  bucket_id     = garage_bucket.test.id
  access_key_id = garage_key.writer.id
  read          = true
  write         = true
}
`, name, maxObjects)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccS3PreCheck(t) },
		ProtoV6ProviderFactories: testAccProviders,
		CheckDestroy:             checkBucketsDestroyed(name),
		Steps: []resource.TestStep{
			{
				Config: config(2),
				Check: func(s *terraform.State) error {
					writer, _ := s3For(s, "garage_key.writer")
					for _, key := range []string{"1", "2"} {
						if err := expectOK("put within the quota", s3Put(writer, name, key, "x")); err != nil {
							return err
						}
					}
					if err := s3Put(writer, name, "3", "x"); err == nil {
						return fmt.Errorf("a third object was accepted with max_objects = 2")
					}
					return nil
				},
			},
			{
				Config: config(3),
				Check: func(s *terraform.State) error {
					writer, _ := s3For(s, "garage_key.writer")
					if err := expectOK("put after raising the quota", s3Put(writer, name, "3", "x")); err != nil {
						return err
					}
					for _, key := range []string{"1", "2", "3"} {
						if err := s3Delete(writer, name, key); err != nil {
							return err
						}
					}
					return nil
				},
			},
		},
	})
}

type httpResult struct {
	status int
	header http.Header
	body   string
}

func doRequest(c *http.Client, req *http.Request) (httpResult, error) {
	resp, err := c.Do(req)
	if err != nil {
		return httpResult{}, err
	}
	defer func() { _ = resp.Body.Close() }()
	data, err := io.ReadAll(resp.Body)
	return httpResult{status: resp.StatusCode, header: resp.Header, body: string(data)}, err
}

func webGet(bucket, path string) (httpResult, error) {
	req, err := http.NewRequest(http.MethodGet, os.Getenv("GARAGE_TEST_WEB_ENDPOINT")+path, nil)
	if err != nil {
		return httpResult{}, err
	}
	req.Host = bucket + ".web.garage.localhost"
	noRedirect := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	return doRequest(noRedirect, req)
}

func TestAccS3_website(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-www")
	website := `
  website = {
    index_document = "index.html"
    error_document = "404.html"
    routing_rules = [
      {
        condition = { key_prefix_equals = "blog/" }
        redirect  = { replace_key_prefix_with = "posts/", http_redirect_code = 301 }
      },
      {
        condition = { http_error_code_returned_equals = 404, key_prefix_equals = "docs/" }
        redirect  = { replace_key_with = "docs/index.html" }
      },
    ]
  }`
	config := func(withWebsite bool) string {
		w := ""
		if withWebsite {
			w = website
		}
		return fmt.Sprintf(`
resource "garage_bucket" "test" {
  global_alias = %[1]q
%[2]s
}

resource "garage_key" "deploy" {
  name = "%[1]s-deploy"
}

resource "garage_bucket_permission" "deploy" {
  bucket_id     = garage_bucket.test.id
  access_key_id = garage_key.deploy.id
  read          = true
  write         = true
}
`, name, w)
	}
	files := map[string]string{"index.html": "home", "404.html": "not found", "docs/index.html": "docs home"}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccS3PreCheck(t) },
		ProtoV6ProviderFactories: testAccProviders,
		CheckDestroy:             checkBucketsDestroyed(name),
		Steps: []resource.TestStep{
			{
				Config: config(true),
				Check: func(s *terraform.State) error {
					deploy, _ := s3For(s, "garage_key.deploy")
					for key, body := range files {
						if err := expectOK("upload "+key, s3Put(deploy, name, key, body)); err != nil {
							return err
						}
					}
					for _, tc := range []struct {
						path, body, location string
						status               int
					}{
						{"/", "home", "", http.StatusOK},
						{"/missing", "not found", "", http.StatusNotFound},
						{"/blog/2026/hello", "", "posts/2026/hello", http.StatusMovedPermanently},
						{"/docs/nope", "", "docs/index.html", http.StatusFound},
					} {
						res, err := webGet(name, tc.path)
						if err != nil {
							return err
						}
						if res.status != tc.status {
							return fmt.Errorf("GET %s: status %d, want %d", tc.path, res.status, tc.status)
						}
						if tc.body != "" && res.body != tc.body {
							return fmt.Errorf("GET %s: body %q, want %q", tc.path, res.body, tc.body)
						}
						if loc := res.header.Get("Location"); tc.location != "" && !strings.HasSuffix(loc, tc.location) {
							return fmt.Errorf("GET %s: Location %q, want a suffix %q", tc.path, loc, tc.location)
						}
					}
					return nil
				},
			},
			{
				Config: config(false),
				Check: func(s *terraform.State) error {
					res, err := webGet(name, "/")
					if err != nil {
						return err
					}
					if res.status == http.StatusOK || res.body == "home" {
						return fmt.Errorf("the bucket is still served after website was removed: %d %q", res.status, res.body)
					}
					deploy, _ := s3For(s, "garage_key.deploy")
					for key := range files {
						if err := s3Delete(deploy, name, key); err != nil {
							return err
						}
					}
					return nil
				},
			},
		},
	})
}

func preflight(bucket, origin, method string) (httpResult, error) {
	req, err := http.NewRequest(http.MethodOptions, os.Getenv("GARAGE_TEST_S3_ENDPOINT")+"/"+bucket+"/upload.bin", nil)
	if err != nil {
		return httpResult{}, err
	}
	req.Header.Set("Origin", origin)
	req.Header.Set("Access-Control-Request-Method", method)
	req.Header.Set("Access-Control-Request-Headers", "content-type")
	return doRequest(http.DefaultClient, req)
}

func preflightAllowed(res httpResult) bool {
	return res.status == http.StatusOK && res.header.Get("Access-Control-Allow-Origin") != ""
}

func TestAccS3_cors(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-cors")
	config := func(rules string) string {
		return fmt.Sprintf(`
resource "garage_bucket" "test" {
  global_alias = %q
  cors_rules   = %s
}
`, name, rules)
	}
	rules := `[
    {
      allowed_origins = ["https://app.example.com"]
      allowed_methods = ["GET", "PUT"]
      allowed_headers = ["*"]
      expose_headers  = ["ETag"]
      max_age_seconds = 3600
    },
  ]`

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccS3PreCheck(t) },
		ProtoV6ProviderFactories: testAccProviders,
		CheckDestroy:             checkBucketsDestroyed(name),
		Steps: []resource.TestStep{
			{
				Config: config(rules),
				Check: func(_ *terraform.State) error {
					res, err := preflight(name, "https://app.example.com", http.MethodPut)
					if err != nil {
						return err
					}
					if res.status != http.StatusOK {
						return fmt.Errorf("allowed preflight: status %d", res.status)
					}
					if got := res.header.Get("Access-Control-Allow-Origin"); got != "https://app.example.com" {
						return fmt.Errorf("Access-Control-Allow-Origin = %q", got)
					}
					if got := res.header.Get("Access-Control-Allow-Methods"); !strings.Contains(got, http.MethodPut) {
						return fmt.Errorf("Access-Control-Allow-Methods = %q", got)
					}
					if got := res.header.Get("Access-Control-Expose-Headers"); got != "ETag" {
						return fmt.Errorf("Access-Control-Expose-Headers = %q", got)
					}
					for _, tc := range [][2]string{{"https://evil.example.com", http.MethodPut}, {"https://app.example.com", http.MethodDelete}} {
						res, err := preflight(name, tc[0], tc[1])
						if err != nil {
							return err
						}
						if preflightAllowed(res) {
							return fmt.Errorf("preflight from %s for %s was allowed", tc[0], tc[1])
						}
					}
					return nil
				},
			},
			{
				Config: config("[]"),
				Check: func(_ *terraform.State) error {
					res, err := preflight(name, "https://app.example.com", http.MethodPut)
					if err != nil {
						return err
					}
					if preflightAllowed(res) {
						return fmt.Errorf("preflight still allowed after cors_rules = []")
					}
					return nil
				},
			},
		},
	})
}
