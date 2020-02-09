package main

import (
	"os"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	common "github.com/mergermarket/cdflow2-config-common"
	"github.com/mergermarket/cdflow2-config-simple-aws/handler"
)

func main() {
	s3Client := s3.New(session.New())
	common.Run(handler.New(s3Client), os.Stdin, os.Stdout, os.Stderr)
}
