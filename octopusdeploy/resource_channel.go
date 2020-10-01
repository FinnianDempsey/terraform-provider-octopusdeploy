package octopusdeploy

import (
	"context"
	"log"

	"github.com/OctopusDeploy/go-octopusdeploy/client"
	"github.com/OctopusDeploy/go-octopusdeploy/model"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceChannel() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceChannelCreate,
		ReadContext:   resourceChannelRead,
		UpdateContext: resourceChannelUpdate,
		DeleteContext: resourceChannelDelete,

		Schema: map[string]*schema.Schema{
			constName: {
				Type:     schema.TypeString,
				Required: true,
			},
			constDescription: {
				Type:     schema.TypeString,
				Optional: true,
			},
			constProjectID: {
				Type:     schema.TypeString,
				Required: true,
			},
			constLifecycleID: {
				Type:     schema.TypeString,
				Optional: true,
			},
			constIsDefault: {
				Type:     schema.TypeBool,
				Optional: true,
			},
			constRule: {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						constVersionRange: {
							Type:     schema.TypeString,
							Optional: true,
						},
						constTag: {
							Type:     schema.TypeString,
							Optional: true,
						},
						constActions: {
							Type:     schema.TypeList,
							Optional: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
					},
				},
			},
		},
	}
}

func resourceChannelCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	channel := buildChannelResource(d)
	diagValidate()

	apiClient := m.(*client.Client)
	resource, err := apiClient.Channels.Add(channel)
	if err != nil {
		return diag.FromErr(createResourceOperationError(errorCreatingChannel, channel.Name, err))
	}

	if isEmpty(resource.ID) {
		log.Println("ID is nil")
	} else {
		d.SetId(resource.ID)
	}

	return nil
}

func buildChannelResource(d *schema.ResourceData) *model.Channel {
	channel := &model.Channel{
		Name:        d.Get(constName).(string),
		Description: d.Get(constDescription).(string),
		ProjectID:   d.Get(constProjectID).(string),
		LifecycleID: d.Get(constLifecycleID).(string),
		IsDefault:   d.Get(constIsDefault).(bool),
	}

	if attr, ok := d.GetOk(constRule); ok {
		tfRules := attr.([]interface{})

		for _, tfrule := range tfRules {
			rule := buildRulesResource(tfrule.(map[string]interface{}))
			channel.Rules = append(channel.Rules, rule)
		}
	}

	return channel
}

func buildRulesResource(tfRule map[string]interface{}) model.ChannelRule {
	rule := model.ChannelRule{
		VersionRange: tfRule[constVersionRange].(string),
		Tag:          tfRule[constTag].(string),
		Actions:      getSliceFromTerraformTypeList(tfRule[constActions]),
	}

	return rule
}

func flattenRules(in []model.ChannelRule) []map[string]interface{} {
	var flattened = make([]map[string]interface{}, len(in))
	for i, v := range in {
		m := make(map[string]interface{})
		m[constVersionRange] = v.VersionRange
		m[constTag] = v.Tag
		m[constActions] = v.Actions

		flattened[i] = m
	}

	return flattened
}

func resourceChannelRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	id := d.Id()
	diagValidate()

	apiClient := m.(*client.Client)
	resource, err := apiClient.Channels.GetByID(id)
	if err != nil {
		return diag.FromErr(createResourceOperationError(errorReadingChannel, id, err))
	}
	if resource == nil {
		d.SetId(constEmptyString)
		return nil
	}

	logResource(constChannel, m)

	d.Set(constName, resource.Name)
	d.Set(constProjectID, resource.ProjectID)
	d.Set(constDescription, resource.Description)
	d.Set(constLifecycleID, resource.LifecycleID)
	d.Set(constIsDefault, resource.IsDefault)
	d.Set(constRule, flattenRules(resource.Rules))

	return nil
}

func resourceChannelUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	channel := buildChannelResource(d)
	channel.ID = d.Id() // set ID so Octopus API knows which channel to update
	diagValidate()

	apiClient := m.(*client.Client)
	resource, err := apiClient.Channels.Update(*channel)
	if err != nil {
		return diag.FromErr(createResourceOperationError(errorUpdatingChannel, d.Id(), err))
	}

	d.SetId(resource.ID)

	return nil
}

func resourceChannelDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	id := d.Id()
	diagValidate()

	apiClient := m.(*client.Client)
	err := apiClient.Channels.DeleteByID(id)
	if err != nil {
		return diag.FromErr(createResourceOperationError(errorDeletingChannel, id, err))
	}

	d.SetId(constEmptyString)
	return nil
}
