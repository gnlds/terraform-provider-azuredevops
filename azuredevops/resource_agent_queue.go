package azuredevops

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	"github.com/microsoft/azure-devops-go-api/azuredevops/taskagent"
	"github.com/microsoft/terraform-provider-azuredevops/azuredevops/utils"
	"github.com/microsoft/terraform-provider-azuredevops/azuredevops/utils/config"
	"github.com/microsoft/terraform-provider-azuredevops/azuredevops/utils/converter"
	"github.com/microsoft/terraform-provider-azuredevops/azuredevops/utils/suppress"
)

const (
	agentPoolID = "agent_pool_id"
	projectID   = "project_id"
)

func resourceAgentQueue() *schema.Resource {
	// Note: there is no update API, so all fields will require a new resource
	return &schema.Resource{
		Create:   resourceAgentQueueCreate,
		Read:     resourceAgentQueueRead,
		Delete:   resourceAgentQueueDelete,
		Importer: importFunc(),
		Schema: map[string]*schema.Schema{
			agentPoolID: {
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: true,
			},
			projectID: {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				ValidateFunc:     validation.NoZeroValues,
				DiffSuppressFunc: suppress.CaseDifference,
			},
		},
	}
}

func resourceAgentQueueCreate(d *schema.ResourceData, m interface{}) error {
	clients := m.(*config.AggregatedClient)
	queue, projectID, err := expandAgentQueue(d)

	referencedPool, err := azureAgentPoolRead(clients, *queue.Pool.Id)
	if err != nil {
		return fmt.Errorf("Error looking up referenced agent pool: %+v", err)
	}

	queue.Name = referencedPool.Name
	createdQueue, err := clients.TaskAgentClient.AddAgentQueue(clients.Ctx, taskagent.AddAgentQueueArgs{
		Queue:              queue,
		Project:            &projectID,
		AuthorizePipelines: converter.Bool(false),
	})

	if err != nil {
		return fmt.Errorf("Error creating agent queue: %+v", err)
	}

	d.SetId(strconv.Itoa(*createdQueue.Id))
	return resourceAgentQueueRead(d, m)
}

func expandAgentQueue(d *schema.ResourceData) (*taskagent.TaskAgentQueue, string, error) {
	queue := &taskagent.TaskAgentQueue{
		Pool: &taskagent.TaskAgentPoolReference{
			Id: converter.Int(d.Get(agentPoolID).(int)),
		},
	}

	if d.Id() != "" {
		id, err := converter.ASCIIToIntPtr(d.Id())
		if err != nil {
			return nil, "", fmt.Errorf("Queue ID was unexpectedly not a valid integer: %+v", err)
		}
		queue.Id = id
	}

	return queue, d.Get(projectID).(string), nil
}

func resourceAgentQueueRead(d *schema.ResourceData, m interface{}) error {
	clients := m.(*config.AggregatedClient)
	queueID, err := converter.ASCIIToIntPtr(d.Id())
	if err != nil {
		return fmt.Errorf("Queue ID was unexpectedly not a valid integer: %+v", err)
	}

	queue, err := clients.TaskAgentClient.GetAgentQueue(clients.Ctx, taskagent.GetAgentQueueArgs{
		QueueId: queueID,
		Project: converter.String(d.Get(projectID).(string)),
	})

	if utils.ResponseWasNotFound(err) {
		d.SetId("")
		return nil
	}

	if err != nil {
		return fmt.Errorf("Error reading the agent queue resource: %+v", err)
	}

	if queue.Pool != nil && queue.Pool.Id != nil {
		d.Set(agentPoolID, *queue.Pool.Id)
	}

	return nil
}

func resourceAgentQueueDelete(d *schema.ResourceData, m interface{}) error {
	clients := m.(*config.AggregatedClient)
	queueID, err := converter.ASCIIToIntPtr(d.Id())
	if err != nil {
		return fmt.Errorf("Queue ID was unexpectedly not a valid integer: %+v", err)
	}

	err = clients.TaskAgentClient.DeleteAgentQueue(clients.Ctx, taskagent.DeleteAgentQueueArgs{
		QueueId: queueID,
		Project: converter.String(d.Get(projectID).(string)),
	})

	if err != nil {
		return fmt.Errorf("Error deleting agent queue: %+v", err)
	}

	d.SetId("")
	return nil
}

func importFunc() *schema.ResourceImporter {
	return &schema.ResourceImporter{
		State: func(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
			id := d.Id()
			parts := strings.SplitN(id, "/", 2)
			if len(parts) != 2 || strings.EqualFold(parts[0], "") || strings.EqualFold(parts[1], "") {
				return nil, fmt.Errorf("unexpected format of ID (%s), expected projectid/resourceId", id)
			}

			_, err := strconv.Atoi(parts[1])
			if err != nil {
				return nil, fmt.Errorf("Agent queue ID (%s) isn't a valid Int", parts[1])
			}

			d.Set(projectID, parts[0])
			d.SetId(parts[1])
			return []*schema.ResourceData{d}, nil
		},
	}
}
