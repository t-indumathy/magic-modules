package storage

import (
	"fmt"

	"github.com/hashicorp/terraform-provider-google-beta/google-beta/tpgiamresource"
	"github.com/hashicorp/terraform-provider-google-beta/google-beta/tpgresource"
	transport_tpg "github.com/hashicorp/terraform-provider-google-beta/google-beta/transport"

	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"google.golang.org/api/cloudresourcemanager/v1"
)

var StorageBucketIamSchema = map[string]*schema.Schema{
	"bucket": {
		Type:             schema.TypeString,
		Required:         true,
		ForceNew:         true,
		DiffSuppressFunc: StorageBucketDiffSuppress,
	},
}

func StorageBucketDiffSuppress(_, old, new string, _ *schema.ResourceData) bool {
	return tpgresource.CompareResourceNames("", old, new, nil)
}

type StorageBucketIamUpdater struct {
	bucket string
	d      tpgresource.TerraformResourceData
	Config *transport_tpg.Config
}

func StorageBucketIamUpdaterProducer(d tpgresource.TerraformResourceData, config *transport_tpg.Config) (tpgiamresource.ResourceIamUpdater, error) {
	values := make(map[string]string)

	if v, ok := d.GetOk("bucket"); ok {
		values["bucket"] = v.(string)
	}

	// We may have gotten either a long or short name, so attempt to parse long name if possible
	m, err := tpgresource.GetImportIdQualifiers([]string{"b/(?P<bucket>[^/]+)", "(?P<bucket>[^/]+)"}, d, config, d.Get("bucket").(string))
	if err != nil {
		return nil, err
	}

	for k, v := range m {
		values[k] = v
	}

	u := &StorageBucketIamUpdater{
		bucket: values["bucket"],
		d:      d,
		Config: config,
	}

	if err := d.Set("bucket", u.GetResourceId()); err != nil {
		return nil, fmt.Errorf("Error setting bucket: %s", err)
	}

	return u, nil
}

func StorageBucketIdParseFunc(d *schema.ResourceData, config *transport_tpg.Config) error {
	values := make(map[string]string)

	m, err := tpgresource.GetImportIdQualifiers([]string{"b/(?P<bucket>[^/]+)", "(?P<bucket>[^/]+)"}, d, config, d.Id())
	if err != nil {
		return err
	}

	for k, v := range m {
		values[k] = v
	}

	u := &StorageBucketIamUpdater{
		bucket: values["bucket"],
		d:      d,
		Config: config,
	}
	if err := d.Set("bucket", u.GetResourceId()); err != nil {
		return fmt.Errorf("Error setting bucket: %s", err)
	}
	d.SetId(u.GetResourceId())
	return nil
}

func (u *StorageBucketIamUpdater) GetResourceIamPolicy() (*cloudresourcemanager.Policy, error) {
	url, err := u.qualifyBucketUrl("iam")
	if err != nil {
		return nil, err
	}

	var obj map[string]interface{}
	url, err = transport_tpg.AddQueryParams(url, map[string]string{"optionsRequestedPolicyVersion": fmt.Sprintf("%d", tpgiamresource.IamPolicyVersion)})
	if err != nil {
		return nil, err
	}

	userAgent, err := tpgresource.GenerateUserAgentString(u.d, u.Config.UserAgent)
	if err != nil {
		return nil, err
	}

	policy, err := transport_tpg.SendRequest(transport_tpg.SendRequestOptions{
		Config:    u.Config,
		Method:    "GET",
		RawURL:    url,
		UserAgent: userAgent,
		Body:      obj,
	})
	if err != nil {
		return nil, errwrap.Wrapf(fmt.Sprintf("Error retrieving IAM policy for %s: {{err}}", u.DescribeResource()), err)
	}

	out := &cloudresourcemanager.Policy{}
	err = tpgresource.Convert(policy, out)
	if err != nil {
		return nil, errwrap.Wrapf("Cannot convert a policy to a resource manager policy: {{err}}", err)
	}

	return out, nil
}

func (u *StorageBucketIamUpdater) SetResourceIamPolicy(policy *cloudresourcemanager.Policy) error {
	json, err := tpgresource.ConvertToMap(policy)
	if err != nil {
		return err
	}

	obj := json

	url, err := u.qualifyBucketUrl("iam")
	if err != nil {
		return err
	}

	userAgent, err := tpgresource.GenerateUserAgentString(u.d, u.Config.UserAgent)
	if err != nil {
		return err
	}

	_, err = transport_tpg.SendRequest(transport_tpg.SendRequestOptions{
		Config:    u.Config,
		Method:    "PUT",
		RawURL:    url,
		UserAgent: userAgent,
		Body:      obj,
		Timeout:   u.d.Timeout(schema.TimeoutCreate),
	})
	if err != nil {
		return errwrap.Wrapf(fmt.Sprintf("Error setting IAM policy for %s: {{err}}", u.DescribeResource()), err)
	}

	return nil
}

func (u *StorageBucketIamUpdater) qualifyBucketUrl(methodIdentifier string) (string, error) {
	urlTemplate := fmt.Sprintf("{{StorageBasePath}}%s/%s", fmt.Sprintf("b/%s", u.bucket), methodIdentifier)
	url, err := tpgresource.ReplaceVars(u.d, u.Config, urlTemplate)
	if err != nil {
		return "", err
	}
	return url, nil
}

func (u *StorageBucketIamUpdater) GetResourceId() string {
	return fmt.Sprintf("b/%s", u.bucket)
}

func (u *StorageBucketIamUpdater) GetMutexKey() string {
	return fmt.Sprintf("iam-storage-bucket-%s", u.GetResourceId())
}

func (u *StorageBucketIamUpdater) DescribeResource() string {
	return fmt.Sprintf("storage bucket %q", u.GetResourceId())
}
