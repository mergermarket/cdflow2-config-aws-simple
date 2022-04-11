package handler

import (
	common "github.com/mergermarket/cdflow2-config-common"
)

func (handler *Handler) PrepareTerraform(request *common.PrepareTerraformRequest, response *common.PrepareTerraformResponse, releaseDir string) error {
	if err := handler.downloadRelease(request); err != nil {
		return err
	}
	return nil
}

// func (h *Handler) PrepareTerraform(request *common.PrepareTerraformRequest, response *common.PrepareTerraformResponse, releaseDir string) error {
// 	team, err := h.getTeam(request.Config["Team"])
// 	if err != nil {
// 		response.Success = false
// 		fmt.Fprintln(h.errorStream, err)
// 	}

// 	releaseAccountCredentialsValue, err := h.ReleaseAccountCredentials.Get()
// 	if err != nil {
// 		response.Success = false
// 		fmt.Fprintln(h.errorStream, err)
// 		return nil
// 	}

// 	response.TerraformBackendType = "s3"
// 	response.TerraformBackendConfig["access_key"] = releaseAccountCredentialsValue.AccessKeyID
// 	response.TerraformBackendConfig["secret_key"] = releaseAccountCredentialsValue.SecretAccessKey
// 	response.TerraformBackendConfig["token"] = releaseAccountCredentialsValue.SessionToken
// 	response.TerraformBackendConfig["region"] = h.getDefaultRegion(request.Config)
// 	response.TerraformBackendConfig["bucket"] = TFStateBucket
// 	// When using a non-default workspace, the state path will be bucket/workspace_key_prefix/workspace_name/key
// 	response.TerraformBackendConfig["workspace_key_prefix"] = fmt.Sprintf("%s/%s", team, request.Component)
// 	response.TerraformBackendConfig["key"] = "terraform.tfstate"
// 	response.TerraformBackendConfig["dynamodb_table"] = fmt.Sprintf("%s-tflocks", team)
// }
