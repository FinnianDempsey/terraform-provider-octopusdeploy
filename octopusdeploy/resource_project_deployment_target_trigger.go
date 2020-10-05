package octopusdeploy

import (
	"context"
	"fmt"
	"log"

	"github.com/OctopusDeploy/go-octopusdeploy/client"
	"github.com/OctopusDeploy/go-octopusdeploy/model"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceProjectDeploymentTargetTrigger() *schema.Resource {
	return &schema.Resource{
		Description:   projectDeploymentTriggerResourceDescription,
		CreateContext: resourceProjectDeploymentTargetTriggerCreate,
		ReadContext:   resourceProjectDeploymentTargetTriggerRead,
		UpdateContext: resourceProjectDeploymentTargetTriggerUpdate,
		DeleteContext: resourceProjectDeploymentTargetTriggerDelete,

		Schema: map[string]*schema.Schema{
			constName: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the trigger.",
			},
			constProjectID: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The project_id of the Project to attach the trigger to.",
			},
			constShouldRedeploy: {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Enable to re-deploy to the deployment targets even if they are already up-to-date with the current deployment.",
			},
			constEventGroups: {
				Type: schema.TypeList,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Optional:    true,
				Description: "Apply event group filters to restrict which deployment targets will actually cause the trigger to fire, and consequently, which deployment targets will be automatically deployed to.",
			},
			constEventCategories: {
				Type: schema.TypeList,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Optional:    true,
				Description: "Apply event category filters to restrict which deployment targets will actually cause the trigger to fire, and consequently, which deployment targets will be automatically deployed to.",
			},
			constRoles: {
				Type: schema.TypeList,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Optional:    true,
				Description: "Apply event role filters to restrict which deployment targets will actually cause the trigger to fire, and consequently, which deployment targets will be automatically deployed to.",
			},
			constEnvironmentIDs: {
				Type: schema.TypeList,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Optional:    true,
				Description: "Apply environment id filters to restrict which deployment targets will actually cause the trigger to fire, and consequently, which deployment targets will be automatically deployed to.",
			},
		},
	}
}

func buildProjectDeploymentTargetTriggerResource(d *schema.ResourceData) (*model.ProjectTrigger, error) {
	name := d.Get(constName).(string)
	projectID := d.Get(constProjectID).(string)
	shouldRedeploy := d.Get(constShouldRedeploy).(bool)

	deploymentTargetTrigger := model.NewProjectDeploymentTargetTrigger(name, projectID, shouldRedeploy, nil, nil, nil)

	if attr, ok := d.GetOk(constEventGroups); ok {
		eventGroups := getSliceFromTerraformTypeList(attr)

		// need to validate here "ValidateFunc is not yet supported on lists or sets."
		validValues := []string{
			"Machine",
			"MachineCritical",
			"MachineAvailableForDeployment",
			"MachineUnavailableForDeployment",
			"MachineHealthChanged",
		}

		if invalidValue, ok := validateAllSliceItemsInSlice(eventGroups, validValues); !ok {
			return nil, fmt.Errorf("Invalid value for event_groups. %s not in %v", invalidValue, validValues)
		}

		deploymentTargetTrigger.AddEventGroups(eventGroups)
	}

	if attr, ok := d.GetOk(constEventCategories); ok {
		eventCategories := getSliceFromTerraformTypeList(attr)

		// need to validate here "ValidateFunc is not yet supported on lists or sets."
		validValues := []string{
			"MachineCleanupFailed",
			"MachineAdded",
			"MachineDeploymentRelatedPropertyWasUpdated",
			"MachineDisabled",
			"MachineEnabled",
			"MachineHealthy",
			"MachineUnavailable",
			"MachineUnhealthy",
			"MachineHasWarnings",
		}

		if invalidValue, ok := validateAllSliceItemsInSlice(eventCategories, validValues); !ok {
			return nil, fmt.Errorf("Invalid value for event_categories. %s not in %v", invalidValue, validValues)
		}

		deploymentTargetTrigger.AddEventCategories(eventCategories)
	}

	if attr, ok := d.GetOk(constRoles); ok {
		deploymentTargetTrigger.Filter.Roles = getSliceFromTerraformTypeList(attr)
	}

	if attr, ok := d.GetOk(constEnvironmentIDs); ok {
		deploymentTargetTrigger.Filter.EnvironmentIDs = getSliceFromTerraformTypeList(attr)
	}

	return deploymentTargetTrigger, nil
}

func resourceProjectDeploymentTargetTriggerCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	projectTrigger, err := buildProjectDeploymentTargetTriggerResource(d)
	if err != nil {
		return diag.FromErr(err)
	}

	diagValidate()

	apiClient := m.(*client.Client)
	resource, err := apiClient.ProjectTriggers.Add(projectTrigger)
	if err != nil {
		return diag.FromErr(createResourceOperationError(errorCreatingProjectTrigger, projectTrigger.Name, err))
	}

	if isEmpty(resource.ID) {
		log.Println("ID is nil")
	} else {
		d.SetId(resource.ID)
	}

	return nil
}

func resourceProjectDeploymentTargetTriggerRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	id := d.Id()
	diagValidate()

	apiClient := m.(*client.Client)
	resource, err := apiClient.ProjectTriggers.GetByID(id)
	if err != nil {
		return diag.FromErr(createResourceOperationError(errorReadingProjectTrigger, id, err))
	}
	if resource == nil {
		d.SetId(constEmptyString)
		return nil
	}

	logResource(constProjectTrigger, m)

	d.Set(constName, resource.Name)
	d.Set(constShouldRedeploy, resource.Action.ShouldRedeployWhenMachineHasBeenDeployedTo)
	d.Set(constEventGroups, resource.Filter.EventGroups)
	d.Set(constEventCategories, resource.Filter.EventCategories)
	d.Set(constRoles, resource.Filter.Roles)
	d.Set(constEnvironmentIDs, resource.Filter.EnvironmentIDs)

	return nil
}

func resourceProjectDeploymentTargetTriggerUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	projectTrigger, err := buildProjectDeploymentTargetTriggerResource(d)
	if err != nil {
		return diag.FromErr(err)
	}
	projectTrigger.ID = d.Id() // set ID so Octopus API knows which project trigger to update

	apiClient := m.(*client.Client)
	resource, err := apiClient.ProjectTriggers.Update(*projectTrigger)
	if err != nil {
		return diag.FromErr(createResourceOperationError(errorUpdatingProjectTrigger, d.Id(), err))
	}

	d.SetId(resource.ID)

	return nil
}

func resourceProjectDeploymentTargetTriggerDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	id := d.Id()
	diagValidate()

	apiClient := m.(*client.Client)
	err := apiClient.ProjectTriggers.DeleteByID(id)
	if err != nil {
		return diag.FromErr(createResourceOperationError(errorDeletingProjectTrigger, id, err))
	}

	d.SetId(constEmptyString)

	return nil
}
