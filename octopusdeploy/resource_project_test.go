package octopusdeploy

import (
	"fmt"
	"testing"

	"github.com/OctopusDeploy/go-octopusdeploy/octopusdeploy"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccOctopusDeployProjectBasic(t *testing.T) {
	const terraformNamePrefix = "octopusdeploy_project.foo"
	const projectName = "Funky Monkey"
	const lifeCycleID = "Lifecycles-1"
	const allowDeploymentsToNoTargets = "true"
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckOctopusDeployProjectDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccProjectBasic(projectName, lifeCycleID, allowDeploymentsToNoTargets),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckOctopusDeployProjectExists(terraformNamePrefix),
					resource.TestCheckResourceAttr(
						terraformNamePrefix, "name", projectName),
					resource.TestCheckResourceAttr(
						terraformNamePrefix, "lifecycle_id", lifeCycleID),
					resource.TestCheckResourceAttr(
						terraformNamePrefix, "allow_deployments_to_no_targets", allowDeploymentsToNoTargets),
				),
			},
		},
	})
}

//nolint:govet
func TestAccOctopusDeployProjectWithUpdate(t *testing.T) {
	return

	const terraformNamePrefix = "octopusdeploy_project.foo"
	const projectName = "Funky Monkey"
	const lifeCycleID = "Lifecycles-1"
	const allowDeploymentsToNoTargets = "true"
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckOctopusDeployProjectDestroy,
		Steps: []resource.TestStep{
			// create project with no description
			{
				Config: testAccProjectBasic(projectName, lifeCycleID, allowDeploymentsToNoTargets),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckOctopusDeployProjectExists(terraformNamePrefix),
					resource.TestCheckResourceAttr(
						terraformNamePrefix, "name", projectName),
					resource.TestCheckResourceAttr(
						terraformNamePrefix, "lifecycle_id", lifeCycleID),
				),
			},
			// create update it with a description + build steps
			{
				Check: resource.ComposeTestCheckFunc(
					testAccCheckOctopusDeployProjectExists(terraformNamePrefix),
					resource.TestCheckResourceAttr(
						terraformNamePrefix, "name", "Project Name"),
					resource.TestCheckResourceAttr(
						terraformNamePrefix, "lifecycle_id", "Lifecycles-1"),
					resource.TestCheckResourceAttr(
						terraformNamePrefix, "description", "My Awesome Description"),
				),
			},
			// update again by remove its description
			{
				Config: testAccProjectBasic(projectName, lifeCycleID, allowDeploymentsToNoTargets),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckOctopusDeployProjectExists(terraformNamePrefix),
					resource.TestCheckResourceAttr(
						terraformNamePrefix, "name", projectName),
					resource.TestCheckResourceAttr(
						terraformNamePrefix, "lifecycle_id", lifeCycleID),
					resource.TestCheckResourceAttr(
						terraformNamePrefix, "description", ""),
				),
			},
		},
	})
}

func testAccProjectBasic(name, lifeCycleID, allowDeploymentsToNoTargets string) string {
	return fmt.Sprintf(`
		resource "octopusdeploy_project_group" "foo" {
			name = "Integration Test Project Group"
		}

		resource "octopusdeploy_project" "foo" {
			name           = "%s"
			lifecycle_id    = "%s"
			project_group_id = "${octopusdeploy_project_group.foo.id}"
			allow_deployments_to_no_targets = "%s"
		}
		`,
		name, lifeCycleID, allowDeploymentsToNoTargets,
	)
}

func testAccCheckOctopusDeployProjectDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*octopusdeploy.Client)

	if err := destroyProjectHelper(s, client); err != nil {
		return err
	}
	return nil
}

func testAccCheckOctopusDeployProjectExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*octopusdeploy.Client)
		if err := existsHelper(s, client); err != nil {
			return err
		}
		return nil
	}
}

func destroyProjectHelper(s *terraform.State, client *octopusdeploy.Client) error {
	for _, r := range s.RootModule().Resources {
		if _, err := client.Project.Get(r.Primary.ID); err != nil {
			if err == octopusdeploy.ErrItemNotFound {
				continue
			}
			return fmt.Errorf("Received an error retrieving project %s", err)
		}
		return fmt.Errorf("Project still exists")
	}
	return nil
}

func existsHelper(s *terraform.State, client *octopusdeploy.Client) error {
	for _, r := range s.RootModule().Resources {
		if r.Type == "octopus_deploy_project" {
			if _, err := client.Project.Get(r.Primary.ID); err != nil {
				return fmt.Errorf("Received an error retrieving project with ID %s: %s", r.Primary.ID, err)
			}
		}
	}
	return nil
}
