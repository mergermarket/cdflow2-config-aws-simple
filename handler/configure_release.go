package handler

import (
	"fmt"

	common "github.com/mergermarket/cdflow2-config-common"
)

// ConfigureRelease runs before the release to provide and check config.
func (h *Handler) ConfigureRelease(request *common.ConfigureReleaseRequest, response *common.ConfigureReleaseResponse) error {
	team, err := h.getTeam(request.Config["team"])
	if err != nil {
		response.Success = false
		fmt.Fprintln(h.ErrorStream, err)
		return nil
	}

	response.MonitoringData["team"] = team
	response.AdditionalMetadata["team"] = team

	if !h.CheckInputConfiguration(request.Config, request.Env) {
		response.Success = false
		return nil
	}

	if !h.CheckAWSResources() {
		response.Success = false
		return nil
	}

	return nil
}

// CheckAWSResources checks that the Release Bucket, Tf State Bucket & Tf Locks Table are present
func (h *Handler) CheckAWSResources() bool {
	problems := 0
	fmt.Fprintf(h.ErrorStream, "%s\n\n", h.styles.au.Underline("Checking AWS resources..."))

	buckets, err := listBuckets(h.getS3Client())
	if err != nil {
		fmt.Fprintf(h.ErrorStream, "%v\n\n", err)
		return false
	}

	if ok, _ := h.handleReleaseBucket(buckets); !ok {
		problems++
	}

	if ok, _ := h.handleTfstateBucket(buckets); !ok {
		problems++
	}

	ok := h.handleTflocksTable()
	if !ok {
		problems++
	}

	// if ok, _ := handler.handleLambdaBucket(response.Env, buckets); !ok {
	// 	warnings++
	// }

	// if ok, _ := handler.handleECRRepository(request.Component, response.Env); !ok {
	// 	warnings++
	// }

	fmt.Fprintln(h.ErrorStream, "")
	if problems > 0 {
		fmt.Fprintf(h.ErrorStream, "To set up AWS resources, please run:\n\n  cdflow2 setup\n\n")
	}

	return problems == 0
}
