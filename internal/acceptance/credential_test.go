package acceptance

import (
	"testing"
)

func TestUcAccCredential(t *testing.T) {
	LoadUcwsEnv(t)
	if IsAws(t) {
		UnityWorkspaceLevel(t, Step{
			Template: `
				resource "databricks_credential" "external" {
					name = "service-cred-{var.RANDOM}"
					aws_iam_role {
						role_arn = "{env.TEST_METASTORE_DATA_ACCESS_ARN}"
					}
					purpose = "SERVICE"
					skip_validation = true
					comment = "Managed by TF"
				}`,
		})
	} else if IsGcp(t) {
		UnityWorkspaceLevel(t, Step{
			// TODO: update purpose to SERVICE when it's released
			Template: `
				resource "databricks_credential" "external" {
					name = "storage-cred-{var.RANDOM}"
					databricks_gcp_service_account {}
					purpose = "STORAGE"
					skip_validation = true
					comment = "Managed by TF"
				}`,
		})
	}
}

func TestAccCredentialOwner(t *testing.T) {
	UnityAccountLevel(t, Step{
		Template: `
			resource "databricks_service_principal" "test_acc_storage_credential_owner" {
				display_name = "test_acc_storage_credential_owner {var.RANDOM}"
			}

			resource "databricks_credential" "test_acc_storage_credential_owner" {
				name = "test_acc_storage_credential_owner-{var.RANDOM}"
				owner = databricks_service_principal.test_acc_storage_credential_owner.application_id
				purpose = "SERVICE"
				aws_iam_role {
					role_arn = "{env.TEST_METASTORE_DATA_ACCESS_ARN}"
				}
			}
		`,
	})
}
