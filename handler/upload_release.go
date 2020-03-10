package handler

import (
	"fmt"
	"io"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	common "github.com/mergermarket/cdflow2-config-common"
)

func (handler *Handler) UploadRelease(request *common.UploadReleaseRequest, response *common.UploadReleaseResponse, version string, config map[string]interface{}) error {
	log.Panicln("uploading...")
	reader, writer := io.Pipe()
	component, _ := config["component"].(string)
	team, _ := config["team"].(string)
	releaseKey := fmt.Sprintf("%s/%s/%s", team, component, version)
	writerDone := make(chan error)
	go func() {
		writerDone <- common.ZipRelease(writer, "/release", component, version)
	}()
	readerDone := make(chan error)
	go func() {
		_, err := s3manager.NewUploaderWithClient(handler.getS3Client()).Upload(&s3manager.UploadInput{
			Bucket: &handler.releaseBucket,
			Key:    &releaseKey,
			Body:   reader,
		})
		readerDone <- err
	}()
	timeout := time.After(60 * time.Second)
	for i := 0; i < 2; i++ {
		select {
		case err := <-writerDone:
			if err != nil {
				return err
			}
		case err := <-readerDone:
			if err != nil {
				return err
			}
		case <-timeout:
			return fmt.Errorf("timeout uploading release to s3 after 60 seconds")
		}
	}
	return nil
}
