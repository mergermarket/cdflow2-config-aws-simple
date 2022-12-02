package handler

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/sts"
	common "github.com/mergermarket/cdflow2-config-common"
)

func (h *Handler) PrepareTerraform(request *common.PrepareTerraformRequest, response *common.PrepareTerraformResponse, releaseDir string) error {
	team, err := h.getTeam(request.Config["team"])
	if err != nil {
		response.Success = false
		fmt.Fprintln(h.ErrorStream, err)
		return nil
	}

	response.MonitoringData["team"] = team

	if !h.CheckInputConfiguration(request.Config, request.Env) {
		response.Success = false
		return nil
	}

	accountID := h.getAccountID()
	if accountID != "" {
		response.MonitoringData["account_id"] = accountID
	}

	if !h.CheckAWSResources() {
		response.Success = false
		return nil
	}

	response.TerraformBackendType = "s3"
	response.TerraformBackendConfig["region"] = h.defaultRegion
	response.TerraformBackendConfig["bucket"] = h.tfstateBucket
	response.TerraformBackendConfig["access_key"] = request.Env["AWS_ACCESS_KEY_ID"]
	response.TerraformBackendConfig["secret_key"] = request.Env["AWS_SECRET_ACCESS_KEY"]
	response.TerraformBackendConfig["token"] = request.Env["AWS_SESSION_TOKEN"]
	// When using a non-default workspace, the state path will be bucket/workspace_key_prefix/workspace_name/key
	response.TerraformBackendConfig["workspace_key_prefix"] = fmt.Sprintf("%s/%s", team, request.Component)
	response.TerraformBackendConfig["key"] = "terraform.tfstate"
	response.TerraformBackendConfig["dynamodb_table"] = h.tflocksTable

	if err := h.AddDeployAccountCredentialsValue(request, team, response.Env); err != nil {
		response.Success = false
		fmt.Fprintln(h.ErrorStream, err)
		return nil
	}

	AddAdditionalEnvironment(request.Env, response.Env)

	s3Client := h.s3Client

	// if request.StateShouldExist != nil {
	// 	statePath := fmt.Sprintf("%s/%s/%s", response.TerraformBackendConfig["workspace_key_prefix"], request.EnvName, response.TerraformBackendConfig["key"])
	// 	if *request.StateShouldExist {
	// 		if err := h.validateStateExists(request, team, statePath, response, s3Client); err != nil {
	// 			response.Success = false
	// 			fmt.Fprintln(h.ErrorStream, err)
	// 			return nil
	// 		}
	// 	}
	// 	if !*request.StateShouldExist {
	// 		if err := h.validateStateDoesNotExist(request, team, statePath, response, s3Client); err != nil {
	// 			response.Success = false
	// 			fmt.Fprintln(h.ErrorStream, err)
	// 			return nil
	// 		}
	// 	}
	// }

	if request.Version == "" {
		return nil
	}
	key := releaseS3Key(team, request.Component, request.Version)
	fmt.Fprintf(h.ErrorStream, "- Downloading release from s3://%s/%s...\n", h.releaseBucket, key)

	getObjectOutput, err := s3Client.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(h.releaseBucket),
		Key:    aws.String(key),
	})
	if err != nil {
		response.Success = false
		fmt.Fprintln(h.ErrorStream, err)
		return nil
	}

	terraformImage, err := h.ReleaseLoader.Load(
		getObjectOutput.Body, request.Component, request.Version, releaseDir,
	)
	if err != nil {
		response.Success = false
		fmt.Fprintln(h.ErrorStream, err)
		return nil
	}

	response.TerraformImage = terraformImage

	return nil
}

func (h *Handler) AddDeployAccountCredentialsValue(request *common.PrepareTerraformRequest, team string, responseEnv map[string]string) error {

	responseEnv["AWS_ACCESS_KEY_ID"] = request.Env["AWS_ACCESS_KEY_ID"]
	responseEnv["AWS_SECRET_ACCESS_KEY"] = request.Env["AWS_SECRET_ACCESS_KEY"]
	responseEnv["AWS_SESSION_TOKEN"] = request.Env["AWS_SESSION_TOKEN"]
	responseEnv["AWS_DEFAULT_REGION"] = request.Env["AWS_DEFAULT_REGION"]

	return nil
}

func (h *Handler) getAccountID() string {
	svc := sts.New(h.awsSession)

	result, err := svc.GetCallerIdentity(&sts.GetCallerIdentityInput{})
	if err != nil {
		fmt.Fprintf(h.ErrorStream, "unable to get aws caller identity: %v", err)
		return ""
	}

	if result.Account != nil {
		return *result.Account
	}

	return ""
}

func AddAdditionalEnvironment(requestEnv map[string]string, responseEnv map[string]string) {
	responseEnv["DD_APP_KEY"] = requestEnv["DATADOG_APP_KEY"]
	responseEnv["DD_API_KEY"] = requestEnv["DATADOG_API_KEY"]
	responseEnv["FASTLY_API_KEY"] = requestEnv["FASTLY_API_KEY"]
	responseEnv["GITHUB_TOKEN"] = requestEnv["GITHUB_TOKEN"]
	responseEnv["MONGODB_ATLAS_PUBLIC_KEY"] = requestEnv["MONGODB_ATLAS_PUBLIC_KEY"]
	responseEnv["MONGODB_ATLAS_PRIVATE_KEY"] = requestEnv["MONGODB_ATLAS_PRIVATE_KEY"]
	responseEnv["JUNOS_PASSWORD"] = requestEnv["JUNOS_PASSWORD"]
}
