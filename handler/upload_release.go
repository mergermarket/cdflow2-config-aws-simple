package handler

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	common "github.com/mergermarket/cdflow2-config-common"
)

func (h *Handler) UploadRelease(request *common.UploadReleaseRequest, response *common.UploadReleaseResponse, configureReleaseRequest *common.ConfigureReleaseRequest, releaseDir string) error {
	log.Println("uploading...")
	team, err := h.getTeam(configureReleaseRequest.Config["team"])
	if err != nil {
		response.Success = false
		fmt.Fprintln(h.ErrorStream, err)
		return nil
	}

	releaseReader, err := h.ReleaseSaver.Save(
		configureReleaseRequest.Component,
		configureReleaseRequest.Version,
		request.TerraformImage,
		releaseDir)
	if err != nil {
		return err
	}
	defer releaseReader.Close()

	releaseKey := releaseS3Key(team, configureReleaseRequest.Component, configureReleaseRequest.Version)
	s3Uploader := s3manager.NewUploaderWithClient(h.getS3Client())
	if _, err := s3Uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(h.releaseBucket),
		Key:    aws.String(releaseKey),
		Body:   releaseReader,
	}); err != nil {
		fmt.Fprintln(h.ErrorStream, "Unable to upload release to S3:", err)
		response.Success = false
		return nil
	}

	fmt.Fprintf(h.ErrorStream, "- Release uploaded to s3://%s/%s\n", h.releaseBucket, releaseKey)

	return nil
}
