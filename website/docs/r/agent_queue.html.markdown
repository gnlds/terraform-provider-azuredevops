---
layout: "azuredevops"
page_title: "AzureDevops: azuredevops_agent_queue"
description: |-
  Manages an agent queue within Azure DevOps project.
---

# azuredevops_agent_queue
Manages an agent queue within Azure DevOps. In the UI, this is equivelant to adding an
Organization defined pool to a project.

## Example Usage

```hcl
resource "azuredevops_project" "p" {
  project_name = "Sample Project"
}

# TODO: Replace hardcoded pool ID with data reference once the following issue is closed:
#   https://github.com/microsoft/terraform-provider-azuredevops/issues/293
resource "azuredevops_agent_queue" "q" {
  project_id             = azuredevops_project.p.id
  agent_pool_id          = 16
  grant_to_all_pipelines = true
}
```

## Argument Reference

The following arguments are supported:

* `project_id` - (Required) The ID of the project in which to create the resource.
* `agent_pool_id` - (Required) The ID of the organization agent pool.
* `grant_to_all_pipelines` - (Required) True if this pool shoud be available to all pipelines, false otherwise. Note: If this a create only field. If this changes in the UI, Terraform will not attempt to revert it.

## Attributes Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The ID of the agent queue reference.

## Relevant Links
* [Azure DevOps Service REST API 5.1 - Agent Queues](https://docs.microsoft.com/en-us/rest/api/azure/devops/distributedtask/queues?view=azure-devops-rest-5.1)

## Import
Azure DevOps Agent Pools can be imported using the project ID and agent queue ID, e.g.

```
terraform import azuredevops_agent_queue.q 44cbf614-4dfd-4032-9fae-87b0da3bec30/1381
```