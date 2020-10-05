package octopusdeploy

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/OctopusDeploy/go-octopusdeploy/client"
	"github.com/OctopusDeploy/go-octopusdeploy/model"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

var mutex = &sync.Mutex{}

func resourceVariable() *schema.Resource {
	return &schema.Resource{
		Description:   variableResourceDescription,
		CreateContext: resourceVariableCreate,
		ReadContext:   resourceVariableRead,
		UpdateContext: resourceVariableUpdate,
		DeleteContext: resourceVariableDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceVariableImport,
		},
		Schema: map[string]*schema.Schema{
			constProjectID: {
				Type:     schema.TypeString,
				Required: true,
			},
			constName: {
				Type:     schema.TypeString,
				Required: true,
			},
			constType: {
				Type:     schema.TypeString,
				Required: true,
				ValidateDiagFunc: validateDiagFunc(validation.StringInSlice([]string{
					"String",
					"Sensitive",
					"Certificate",
					"AmazonWebServicesAccount",
					"AzureAccount",
				}, false)),
			},
			constValue: {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{constSensitiveValue},
			},
			constSensitiveValue: {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{constValue},
				Sensitive:     true,
			},
			constDescription: {
				Type:     schema.TypeString,
				Optional: true,
			},
			constScope: schemaVariableScope,
			constIsSensitive: {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			constPrompt: {
				Type:     schema.TypeSet,
				MaxItems: 1,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						constLabel: {
							Type:     schema.TypeString,
							Optional: true,
						},
						constDescription: {
							Type:     schema.TypeString,
							Optional: true,
						},
						constRequired: {
							Type:     schema.TypeBool,
							Optional: true,
						},
					},
				},
			},
			constPGPKey: {
				Type:      schema.TypeString,
				Optional:  true,
				ForceNew:  true,
				Sensitive: true,
			},
			constKeyFingerprint: {
				Type:     schema.TypeString,
				Computed: true,
			},
			constEncryptedValue: {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceVariableImport(ctx context.Context, d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	importStrings := strings.Split(d.Id(), ":")
	if len(importStrings) != 2 {
		return nil, fmt.Errorf("octopusdeploy_variable import must be in the form of ProjectID:VariableID (e.g. Projects-62:0906031f-68ba-4a15-afaa-657c1564e07b")
	}

	d.Set(constProjectID, importStrings[0])
	d.SetId(importStrings[1])

	return []*schema.ResourceData{d}, nil
}

func resourceVariableRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	id := d.Id()
	projectID := d.Get(constProjectID).(string)

	diagValidate()

	apiClient := m.(*client.Client)
	resource, err := apiClient.Variables.GetByID(projectID, id)
	if err != nil {
		return diag.FromErr(createResourceOperationError(errorReadingVariable, id, err))
	}
	if resource == nil {
		d.SetId(constEmptyString)
		return nil
	}

	logResource(constVariable, m)

	d.Set(constName, resource.Name)
	d.Set(constType, resource.Type)

	isSensitive := d.Get(constIsSensitive).(bool)
	if isSensitive {
		d.Set(constValue, nil)
	} else {
		d.Set(constValue, resource.Value)
	}

	d.Set(constDescription, resource.Description)

	return nil
}

func buildVariableResource(d *schema.ResourceData) *model.Variable {
	varName := d.Get(constName).(string)
	varType := d.Get(constType).(string)

	var varDesc, varValue string
	var varSensitive bool

	if varDescInterface, ok := d.GetOk(constDescription); ok {
		varDesc = varDescInterface.(string)
	}

	if varSensitiveInterface, ok := d.GetOk(constIsSensitive); ok {
		varSensitive = varSensitiveInterface.(bool)
	}

	if varSensitive {
		varValue = d.Get(constSensitiveValue).(string)
	} else {
		varValue = d.Get(constValue).(string)
	}

	varScopeInterface := tfVariableScopetoODVariableScope(d)

	newVar := model.NewVariable(varName, varType, varValue, varDesc, varScopeInterface, varSensitive)

	varPrompt, ok := d.GetOk(constPrompt)
	if ok {
		tfPromptSettings := varPrompt.(*schema.Set)
		if len(tfPromptSettings.List()) == 1 {
			tfPromptList := tfPromptSettings.List()[0].(map[string]interface{})
			newPrompt := model.VariablePromptOptions{
				Description: tfPromptList[constDescription].(string),
				Label:       tfPromptList[constLabel].(string),
				Required:    tfPromptList[constRequired].(bool),
			}
			newVar.Prompt = &newPrompt
		}
	}

	return newVar
}

func resourceVariableCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	mutex.Lock()
	defer mutex.Unlock()
	if err := validateVariable(ctx, d); err != nil {
		return err
	}

	diagValidate()

	projID := d.Get(constProjectID).(string)
	newVariable := buildVariableResource(d)

	apiClient := m.(*client.Client)
	tfVar, err := apiClient.Variables.AddSingle(projID, newVariable)
	if err != nil {
		return diag.FromErr(createResourceOperationError(errorCreatingVariable, newVariable.Name, err))
	}

	for _, v := range tfVar.Variables {
		if v.Name == newVariable.Name && v.Type == newVariable.Type && (v.IsSensitive || v.Value == newVariable.Value) && v.Description == newVariable.Description && v.IsSensitive == newVariable.IsSensitive {
			scopeMatches, _, err := apiClient.Variables.MatchesScope(v.Scope, newVariable.Scope)
			if err != nil {
				return diag.FromErr(err)
			}
			if scopeMatches {
				d.SetId(v.ID)
				return nil
			}
		}
	}

	d.SetId(constEmptyString)
	return diag.Errorf("unable to locate variable in project %s", projID)
}

func resourceVariableUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	mutex.Lock()
	defer mutex.Unlock()

	diagValidate()

	if err := validateVariable(ctx, d); err != nil {
		return err
	}

	tfVar := buildVariableResource(d)
	tfVar.ID = d.Id() // set ID so Octopus API knows which variable to update
	projID := d.Get(constProjectID).(string)

	apiClient := m.(*client.Client)
	updatedVars, err := apiClient.Variables.UpdateSingle(projID, tfVar)
	if err != nil {
		return diag.FromErr(createResourceOperationError(errorUpdatingVariable, d.Id(), err))
	}

	for _, v := range updatedVars.Variables {
		if v.Name == tfVar.Name && v.Type == tfVar.Type && (v.IsSensitive || v.Value == tfVar.Value) && v.Description == tfVar.Description && v.IsSensitive == tfVar.IsSensitive {
			scopeMatches, _, _ := apiClient.Variables.MatchesScope(v.Scope, tfVar.Scope)
			if scopeMatches {
				d.SetId(v.ID)
				return nil
			}
		}
	}

	d.SetId(constEmptyString)
	return diag.Errorf("unable to locate variable in project %s", projID)
}

func resourceVariableDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	mutex.Lock()
	defer mutex.Unlock()

	diagValidate()

	projID := d.Get(constProjectID).(string)
	variableID := d.Id()

	apiClient := m.(*client.Client)
	_, err := apiClient.Variables.DeleteSingle(projID, variableID)
	if err != nil {
		return diag.FromErr(createResourceOperationError(errorDeletingVariable, variableID, err))
	}

	d.SetId(constEmptyString)
	return nil
}

// Validating is done in its own function as we need to compare options once the entire
// schema has been parsed, which as far as I can tell we can't do in a normal validation
// function.
func validateVariable(ctx context.Context, d *schema.ResourceData) diag.Diagnostics {
	tfSensitive := d.Get(constIsSensitive).(bool)
	tfType := d.Get(constType).(string)

	if tfSensitive && tfType != "Sensitive" {
		return diag.Errorf("when is_sensitive is set to true, type needs to be 'Sensitive'")
	}

	if !tfSensitive && tfType == "Sensitive" {
		return diag.Errorf("when type is set to 'Sensitive', is_sensitive needs to be true")
	}

	return nil
}
