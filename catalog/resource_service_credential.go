package catalog

import (
	"context"

	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/databricks/terraform-provider-databricks/catalog/bindings"
	"github.com/databricks/terraform-provider-databricks/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

type ServiceCredentialInfo struct {
	Name                        string                                       `json:"name" tf:"force_new"`
	Owner                       string                                       `json:"owner,omitempty" tf:"computed"`
	Comment                     string                                       `json:"comment,omitempty"`
	Aws                         *catalog.AwsIamRoleResponse                  `json:"aws_iam_role,omitempty" tf:"group:access"`
	Azure                       *catalog.AzureServicePrincipal               `json:"azure_service_principal,omitempty" tf:"group:access"`
	AzMI                        *catalog.AzureManagedIdentityResponse        `json:"azure_managed_identity,omitempty" tf:"group:access"`
	GcpSAKey                    *GcpServiceAccountKey                        `json:"gcp_service_account_key,omitempty" tf:"group:access"`
	DatabricksGcpServiceAccount *catalog.DatabricksGcpServiceAccountResponse `json:"databricks_gcp_service_account,omitempty" tf:"computed"`
	CloudflareApiToken          *catalog.CloudflareApiToken                  `json:"cloudflare_api_token,omitempty" tf:"group:access"`
	MetastoreID                 string                                       `json:"metastore_id,omitempty" tf:"computed"`
	ReadOnly                    bool                                         `json:"read_only,omitempty"`
	SkipValidation              bool                                         `json:"skip_validation,omitempty"`
	IsolationMode               string                                       `json:"isolation_mode,omitempty" tf:"computed"`
	Purpose                     string                                       `json:"purpose" tf:"required"`
}

var serviceCredentialSchema = common.StructToSchema(ServiceCredentialInfo{},
	func(m map[string]*schema.Schema) map[string]*schema.Schema {
		m["service_credential_id"] = &schema.Schema{
			Type:     schema.TypeString,
			Computed: true,
		}
		common.MustSchemaPath(m, "databricks_gcp_service_account", "email").Computed = true
		common.MustSchemaPath(m, "databricks_gcp_service_account", "credential_id").Computed = true
		return adjustDataAccessSchema(m)
	})

func ResourceServiceCredential() common.Resource {
	return common.Resource{
		Schema: serviceCredentialSchema,
		Create: func(ctx context.Context, d *schema.ResourceData, c *common.DatabricksClient) error {
			metastoreId := d.Get("metastore_id").(string)

			var create catalog.CreateCredentialRequest
			var update catalog.UpdateCredentialRequest
			common.DataToStructPointer(d, serviceCredentialSchema, &create)
			common.DataToStructPointer(d, serviceCredentialSchema, &update)
			update.NameArg = d.Get("name").(string)
			create.Purpose = "SERVICE"

			return c.AccountOrWorkspaceRequest(func(acc *databricks.AccountClient) error {
				cred, err := acc.Credentials.CreateCredential(ctx, catalog.AccountsCreateCredentialRequest{
					MetastoreId: metastoreId,
					CredentialInfo: &create,
				})
				if err != nil {
					return err
				}
				d.SetId(cred.CredentialInfo.Name)

				// Update owner or isolation mode if it is provided
				if !updateRequired(d, []string{"owner", "isolation_mode"}) {
					return nil
				}

				update.NameArg = d.Id()
				_, err = acc.Credentials.UpdateCredential(ctx, catalog.AccountsUpdateCredentialRequest{
					CredentialInfo: &update,
					MetastoreId:    metastoreId,
					CredentialName: cred.CredentialInfo.Name,
				})
				if err != nil {
					return err
				}
				return nil
			}, func(w *databricks.WorkspaceClient) error {
				err := validateMetastoreId(ctx, w, d.Get("metastore_id").(string))
				if err != nil {
					return err
				}
				cred, err := w.Credentials.CreateCredential(ctx, create)
				if err != nil {
					return err
				}
				d.SetId(cred.Name)

				// Update owner or isolation mode if it is provided
				if !updateRequired(d, []string{"owner", "isolation_mode"}) {
					return nil
				}

				update.NameArg = d.Id()
				_, err = w.Credentials.UpdateCredential(ctx, update)
				if err != nil {
					return err
				}
				// Bind the current workspace if the credential is isolated, otherwise the read will fail
				return bindings.AddCurrentWorkspaceBindings(ctx, d, w, cred.Name, catalog.UpdateBindingsSecurableTypeCredential)
			})
		},
		Read: func(ctx context.Context, d *schema.ResourceData, c *common.DatabricksClient) error {
			return c.AccountOrWorkspaceRequest(func(acc *databricks.AccountClient) error {
				cred, err := acc.Credentials.GetCredentialByNameArg(ctx, d.Id())
				if err != nil {
					return err
				}
				// azure client secret is sensitive, so we need to preserve it
				var credOrig catalog.CredentialInfo
				common.DataToStructPointer(d, serviceCredentialSchema, &credOrig)
				if credOrig.AzureServicePrincipal != nil {
					if credOrig.AzureServicePrincipal.ClientSecret != "" {
						cred.AzureServicePrincipal.ClientSecret = credOrig.AzureServicePrincipal.ClientSecret
					}
				}
				d.Set("credential_id", cred.Id)
				return common.StructToData(cred, serviceCredentialSchema, d)
			}, func(w *databricks.WorkspaceClient) error {
				cred, err := w.Credentials.GetCredentialByNameArg(ctx, d.Id())
				if err != nil {
					return err
				}
				// azure client secret is sensitive, so we need to preserve it
				var credOrig catalog.CredentialInfo
				common.DataToStructPointer(d, serviceCredentialSchema, &credOrig)
				if credOrig.AzureServicePrincipal != nil {
					if credOrig.AzureServicePrincipal.ClientSecret != "" {
						cred.AzureServicePrincipal.ClientSecret = credOrig.AzureServicePrincipal.ClientSecret
					}
				}
				d.Set("credential_id", cred.Id)
				return common.StructToData(cred, serviceCredentialSchema, d)
			})
		},
		Update: func(ctx context.Context, d *schema.ResourceData, c *common.DatabricksClient) error {
			var update catalog.UpdateCredentialRequest
			force := d.Get("force_update").(bool)
			common.DataToStructPointer(d, serviceCredentialSchema, &update)
			update.NameArg = d.Id()
			update.Force = force

			return c.AccountOrWorkspaceRequest(func(acc *databricks.AccountClient) error {
				if d.HasChange("owner") {
					_, err := acc.Credentials.UpdateCredential(ctx, catalog.AccountsUpdateCredentialRequest{
						CredentialInfo: &catalog.UpdateCredentialRequest{
							NameArg: update.NameArg,
							Owner:   update.Owner,
						},
						MetastoreId:    d.Get("metastore_id").(string),
						CredentialName: d.Id(),
					})
					if err != nil {
						return err
					}
				}

				if !d.HasChangeExcept("owner") {
					return nil
				}

				if d.HasChange("read_only") {
					update.ForceSendFields = append(update.ForceSendFields, "ReadOnly")
				}
				update.Owner = ""
				_, err := acc.Credentials.UpdateCredential(ctx, catalog.AccountsUpdateCredentialRequest{
					CredentialInfo: &update,
					MetastoreId:    d.Get("metastore_id").(string),
					CredentialName: d.Id(),
				})
				if err != nil {
					if d.HasChange("owner") {
						// Rollback
						old, new := d.GetChange("owner")
						_, rollbackErr := acc.Credentials.UpdateCredential(ctx, catalog.AccountsUpdateCredentialRequest{
							CredentialInfo: &catalog.UpdateCredentialRequest{
								NameArg: update.NameArg,
								Owner:   old.(string),
							},
							MetastoreId:    d.Get("metastore_id").(string),
							CredentialName: d.Id(),
						})
						if rollbackErr != nil {
							return common.OwnerRollbackError(err, rollbackErr, old.(string), new.(string))
						}
					}
					return err
				}
				return nil
			}, func(w *databricks.WorkspaceClient) error {
				err := validateMetastoreId(ctx, w, d.Get("metastore_id").(string))
				if err != nil {
					return err
				}
				if d.HasChange("owner") {
					_, err := w.Credentials.UpdateCredential(ctx, catalog.UpdateCredentialRequest{
						NameArg: update.NameArg,
						Owner:   update.Owner,
					})
					if err != nil {
						return err
					}
				}

				if !d.HasChangeExcept("owner") {
					return nil
				}

				if d.HasChange("read_only") {
					update.ForceSendFields = append(update.ForceSendFields, "ReadOnly")
				}
				update.Owner = ""
				_, err = w.Credentials.UpdateCredential(ctx, update)
				if err != nil {
					if d.HasChange("owner") {
						// Rollback
						old, new := d.GetChange("owner")
						_, rollbackErr := w.Credentials.UpdateCredential(ctx, catalog.UpdateCredentialRequest{
							NameArg: update.NameArg,
							Owner:   old.(string),
						})
						if rollbackErr != nil {
							return common.OwnerRollbackError(err, rollbackErr, old.(string), new.(string))
						}
					}
					return err
				}
				// Bind the current workspace if the credential is isolated, otherwise the read will fail
				return bindings.AddCurrentWorkspaceBindings(ctx, d, w, update.NameArg, catalog.UpdateBindingsSecurableTypeCredential)
			})
		},
		Delete: func(ctx context.Context, d *schema.ResourceData, c *common.DatabricksClient) error {
			force := d.Get("force_destroy").(bool)
			return c.AccountOrWorkspaceRequest(func(acc *databricks.AccountClient) error {
				return acc.Credentials.DeleteCredential(ctx, catalog.DeleteAccountCredentialRequest{
					Force:          force,
					CredentialName: d.Id(),
					MetastoreId:    d.Get("metastore_id").(string),
				})
			}, func(w *databricks.WorkspaceClient) error {
				err := validateMetastoreId(ctx, w, d.Get("metastore_id").(string))
				if err != nil {
					return err
				}
				return w.Credentials.DeleteCredential(ctx, catalog.DeleteCredentialRequest{
					Force: force,
					Name:  d.Id(),
				})
			})
		},
	}
}
