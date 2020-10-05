package octopusdeploy

import (
	"context"
	"log"

	"github.com/OctopusDeploy/go-octopusdeploy/client"
	"github.com/OctopusDeploy/go-octopusdeploy/enum"
	"github.com/OctopusDeploy/go-octopusdeploy/model"
	uuid "github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourceAzureServicePrincipal() *schema.Resource {
	validateSchema()

	log.Println("Hello")
	schemaMap := getCommonAccountsSchema()

	schemaMap[constClientID] = &schema.Schema{
		Type:     schema.TypeString,
		Required: true,
	}
	schemaMap[constTenantID] = &schema.Schema{
		Type:     schema.TypeString,
		Required: true,
	}
	schemaMap[constSubscriptionNumber] = &schema.Schema{
		Type: schema.TypeString,
		//Computed:     true,
		Required:         true,
		ValidateDiagFunc: validateDiagFunc(validation.IsUUID),
	}
	schemaMap[constKey] = &schema.Schema{
		Type:      schema.TypeString,
		Required:  true,
		Sensitive: true,
	}
	schemaMap[constAzureEnvironment] = &schema.Schema{
		Type:     schema.TypeString,
		Optional: true,
	}
	schemaMap[constResourceManagementEndpointBaseURI] = &schema.Schema{
		Type:     schema.TypeString,
		Optional: true,
	}
	schemaMap[constActiveDirectoryEndpointBaseURI] = &schema.Schema{
		Type:     schema.TypeString,
		Optional: true,
	}

	return &schema.Resource{
		Description:   azureAccountResourceDescription,
		CreateContext: resourceAzureServicePrincipalCreate,
		ReadContext:   resourceAzureServicePrincipalRead,
		UpdateContext: resourceAzureServicePrincipalUpdate,
		DeleteContext: resourceAccountDeleteCommon,
		Schema:        schemaMap,
	}
}

func buildAzureServicePrincipalResource(d *schema.ResourceData) (*model.Account, error) {
	name := d.Get(constName).(string)

	password := d.Get(constKey).(string)
	if isEmpty(password) {
		log.Println("Key is nil. Must add in a password")
	}

	secretKey := model.NewSensitiveValue(password)

	applicationID, err := uuid.Parse(d.Get(constClientID).(string))
	if err != nil {
		log.Println(err)
		return nil, err
	}

	tenantID, err := uuid.Parse(d.Get(constTenantID).(string))
	if err != nil {
		log.Println(err)
		return nil, err
	}

	subscriptionID, err := uuid.Parse(d.Get(constSubscriptionNumber).(string))
	if err != nil {
		log.Println(err)
		return nil, err
	}

	account, err := model.NewAzureServicePrincipalAccount(name, subscriptionID, tenantID, applicationID, secretKey)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	// Optional Fields
	if v, ok := d.GetOk(constDescription); ok {
		account.Description = v.(string)
	}

	if v, ok := d.GetOk(constEnvironments); ok {
		account.EnvironmentIDs = getSliceFromTerraformTypeList(v)
	}

	if v, ok := d.GetOk(constTenantedDeploymentParticipation); ok {
		account.TenantedDeploymentParticipation, _ = enum.ParseTenantedDeploymentMode(v.(string))
	}

	if v, ok := d.GetOk(constTenantTags); ok {
		account.TenantTags = getSliceFromTerraformTypeList(v)
	}

	if v, ok := d.GetOk(constTenants); ok {
		account.TenantIDs = getSliceFromTerraformTypeList(v)
	}

	if v, ok := d.GetOk(constResourceManagementEndpointBaseURI); ok {
		account.ResourceManagementEndpointBase = v.(string)
	}

	if v, ok := d.GetOk(constActiveDirectoryEndpointBaseURI); ok {
		account.ActiveDirectoryEndpointBase = v.(string)
	}

	err = account.Validate()
	if err != nil {
		return nil, err
	}

	return account, nil
}

func resourceAzureServicePrincipalCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	account, err := buildAzureServicePrincipalResource(d)
	if err != nil {
		log.Println(err)
		return diag.FromErr(err)
	}

	diagValidate()

	apiClient := m.(*client.Client)
	resource, err := apiClient.Accounts.Add(account)
	if err != nil {
		return diag.FromErr(createResourceOperationError(errorCreatingAzureServicePrincipal, account.Name, err))
	}

	if isEmpty(resource.ID) {
		log.Println("ID is nil")
	} else {
		d.SetId(resource.ID)
	}

	return nil
}

func resourceAzureServicePrincipalRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	id := d.Id()

	diagValidate()

	apiClient := m.(*client.Client)
	resource, err := apiClient.Accounts.GetByID(id)
	if err != nil {
		return diag.FromErr(createResourceOperationError(errorReadingAzureServicePrincipal, id, err))
	}
	if resource == nil {
		d.SetId(constEmptyString)
		return nil
	}

	logResource(constAccount, m)

	d.Set(constName, resource.Name)
	d.Set(constDescription, resource.Description)
	d.Set(constEnvironments, resource.EnvironmentIDs)
	d.Set(constTenantedDeploymentParticipation, resource.TenantedDeploymentParticipation.String())
	d.Set(constTenantTags, resource.TenantTags)
	d.Set(constClientID, resource.ApplicationID)
	d.Set(constTenantID, resource.TenantIDs)
	d.Set(constSubscriptionNumber, resource.SubscriptionID)
	d.Set(constKey, resource.Password)
	d.Set(constAzureEnvironment, resource.AzureEnvironment)
	d.Set(constResourceManagementEndpointBaseURI, resource.ResourceManagementEndpointBase)
	d.Set(constActiveDirectoryEndpointBaseURI, resource.ActiveDirectoryEndpointBase)

	return nil
}

func resourceAzureServicePrincipalUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	diagValidate()
	account, err := buildAzureServicePrincipalResource(d)
	if err != nil {
		return diag.FromErr(err)
	}

	account.ID = d.Id() // set ID so Octopus API knows which account to update

	apiClient := m.(*client.Client)
	resource, err := apiClient.Accounts.Update(*account)
	if err != nil {
		return diag.FromErr(createResourceOperationError(errorUpdatingAzureServicePrincipal, d.Id(), err))
	}

	d.SetId(resource.ID)

	return nil
}
