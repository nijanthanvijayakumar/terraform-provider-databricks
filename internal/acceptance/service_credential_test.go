package acceptance

import (
	"testing"

	"github.com/databricks/terraform-provider-databricks/acceptance"
	"github.com/databricks/terraform-provider-databricks/acceptance/qa"
)

func TestAccServiceCredential(t *testing.T) {
	acceptance.Test(t, []acceptance.Step{
		{
			Template: `
			resource "databricks_service_credential" "this" {
				name = "test-service-credential"
				aws_iam_role {
					role_arn = "arn:aws:iam::123456789012:role/test-role"
				}
				comment = "Test service credential"
			}`,
		},
	})
}

func TestAccServiceCredential_Isolated(t *testing.T) {
	acceptance.Test(t, []acceptance.Step{
		{
			Template: `
			resource "databricks_service_credential" "this" {
				name = "test-service-credential-isolated"
				aws_iam_role {
					role_arn = "arn:aws:iam::123456789012:role/test-role"
				}
				comment = "Test isolated service credential"
				isolation_mode = "ISOLATION_MODE_ISOLATED"
			}`,
		},
	})
}

func TestAccServiceCredential_Azure(t *testing.T) {
	acceptance.Test(t, []acceptance.Step{
		{
			Template: `
			resource "databricks_service_credential" "this" {
				name = "test-service-credential-azure"
				azure_managed_identity {
					access_connector_id = "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-name/providers/Microsoft.Databricks/accessConnectors/connector-name"
				}
				comment = "Test Azure service credential"
			}`,
		},
	})
}

func TestAccServiceCredential_GCP(t *testing.T) {
	acceptance.Test(t, []acceptance.Step{
		{
			Template: `
			resource "databricks_service_credential" "this" {
				name = "test-service-credential-gcp"
				databricks_gcp_service_account {}
				comment = "Test GCP service credential"
			}`,
		},
	})
}
