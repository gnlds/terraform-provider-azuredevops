package azuredevops

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	"github.com/microsoft/azure-devops-go-api/azuredevops/taskagent"
	"github.com/microsoft/terraform-provider-azuredevops/azuredevops/utils"
	"github.com/microsoft/terraform-provider-azuredevops/azuredevops/utils/config"
	"github.com/microsoft/terraform-provider-azuredevops/azuredevops/utils/converter"
	"github.com/microsoft/terraform-provider-azuredevops/azuredevops/utils/suppress"
	"strconv"
	"strings"
)

const (
	agentPoolID         = "agent_pool_id"
	grantToAllPipelines = "grant_to_all_pipelines"
	projectID           = "project_id"
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
			grantToAllPipelines: {
				Type: schema.TypeBool,
				// This is a create only field. Ignore all changes
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool { return true },
				Default:          true,
				Optional:         true,
				// Needed to make Terraform happy
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

type agentQueueArgs struct {
	agentQueue         *taskagent.TaskAgentQueue
	authorizePipelines bool
	projectID          string
}

func resourceAgentQueueCreate(d *schema.ResourceData, m interface{}) error {
	clients := m.(*config.AggregatedClient)
	queueArgs, err := expandAgentQueue(d)

	referencedPool, err := azureAgentPoolRead(clients, *queueArgs.agentQueue.Pool.Id)
	if err != nil {
		return fmt.Errorf("Error looking up referenced agent pool: %+v", err)
	}

	queueArgs.agentQueue.Name = referencedPool.Name
	createdQueue, err := clients.TaskAgentClient.AddAgentQueue(clients.Ctx, taskagent.AddAgentQueueArgs{
		Queue:              queueArgs.agentQueue,
		Project:            &queueArgs.projectID,
		AuthorizePipelines: &queueArgs.authorizePipelines,
	})

	if err != nil {
		return fmt.Errorf("Error creating agent queue: %+v", err)
	}

	d.SetId(strconv.Itoa(*createdQueue.Id))
	return resourceAgentQueueRead(d, m)
}

func expandAgentQueue(d *schema.ResourceData) (*agentQueueArgs, error) {
	queue := &taskagent.TaskAgentQueue{
		Pool: &taskagent.TaskAgentPoolReference{
			Id: converter.Int(d.Get(agentPoolID).(int)),
		},
	}

	if d.Id() != "" {
		id, err := asciiToIntPtr(d.Id())
		if err != nil {
			return nil, fmt.Errorf("Queue ID was unexpectedly not a valid integer: %+v", err)
		}
		queue.Id = id
	}

	return &agentQueueArgs{
		authorizePipelines: d.Get(grantToAllPipelines).(bool),
		projectID:          d.Get(projectID).(string),
		agentQueue:         queue,
	}, nil
}

func asciiToIntPtr(value string) (*int, error) {
	i, err := strconv.Atoi(value)
	if err != nil {
		return nil, err
	}
	return converter.Int(i), nil
}

func resourceAgentQueueRead(d *schema.ResourceData, m interface{}) error {
	clients := m.(*config.AggregatedClient)
	queueID, err := asciiToIntPtr(d.Id())
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

	if queue.Pool != nil && queue.Pool.Id != nil {
		d.Set(agentPoolID, *queue.Pool.Id)
	}

	return nil
}

func resourceAgentQueueDelete(d *schema.ResourceData, m interface{}) error {
	clients := m.(*config.AggregatedClient)
	queueID, err := asciiToIntPtr(d.Id())
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
