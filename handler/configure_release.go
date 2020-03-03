package handler

import (
	"fmt"

	common "github.com/mergermarket/cdflow2-config-common"
)

// ConfigureRelease runs before the release to provide and check config.
func (handler *Handler) ConfigureRelease(request *common.ConfigureReleaseRequest, response *common.ConfigureReleaseResponse) error {
	if !handler.checkInputConfiguration(request.Config, request.Env, response.Env) {
		response.Success = false
		return nil
	}

	if !handler.checkAWSResources(request, response) {
		response.Success = false
		return nil
	}

	return nil
}

func (handler *Handler) checkAWSResources(request *common.ConfigureReleaseRequest, response *common.ConfigureReleaseResponse) bool {
	problems := 0
	fmt.Fprintf(handler.errorStream, "%s\n\n", handler.styles.au.Underline("Checking AWS resources..."))

	buckets, err := listBuckets(handler.getS3Client())
	if err != nil {
		fmt.Fprintf(handler.errorStream, "%v\n\n", err)
		return false
	}

	if ok, _ := handler.handleReleaseBucket(buckets); ok {
		problems++
	}

	if ok, _ := handler.handleTfstateBucket(buckets); ok {
		problems++
	}

	warnings := 0

	ok, err := handler.handleTflocksTable()
	if err != nil {
		fmt.Fprintf(handler.errorStream, "%v\n\n", err)
		return false
	} else if !ok {
		warnings++
	}

	if ok, _ := handler.handleLambdaBucket(response.Env, buckets); !ok {
		warnings++
	}

	if ok, _ := handler.handleECRRepository(request.Component, response.Env); !ok {
		warnings++
	}

	fmt.Fprintln(handler.errorStream, "")
	if problems > 0 || warnings > 0 {
		fmt.Fprintf(handler.errorStream, "To set up AWS resources, please run:\n\n  cdflow2 setup\n\n")
	}

	return problems == 0
}
