package handler

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"math/big"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/aws/aws-sdk-go/service/ecr/ecriface"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/logrusorgru/aurora"
	common "github.com/mergermarket/cdflow2-config-common"
)

type styles struct {
	au           aurora.Aurora
	tick         string
	cross        string
	warningCross string
}

func initStyles() *styles {
	au := aurora.NewAurora(true)
	result := styles{
		au:           au,
		cross:        fmt.Sprintf("%s", au.Red("✖")),
		warningCross: fmt.Sprintf("%s", au.Yellow("✖")),
		tick:         fmt.Sprintf("%s", au.Green("✔")),
	}
	return &result
}

// Handler handles config requests.
type Handler struct {
	s3Client       s3iface.S3API
	dynamoDBClient dynamodbiface.DynamoDBAPI
	ecrClient      ecriface.ECRAPI
	awsSession     *session.Session
	defaultRegion  string
	releaseDir     string
	releaseBucket  string
	tfstateBucket  string
	tflocksTable   string
	lambdaBucket   string
	inputStream    io.Reader
	outputStream   io.Writer
	errorStream    io.Writer
	styles         *styles
}

// Opts are the options for creating a new handler.
type Opts struct {
	S3Client       s3iface.S3API
	DynamoDBClient dynamodbiface.DynamoDBAPI
	ECRClient      ecriface.ECRAPI
	ReleaseDir     string
	InputStream    io.Reader
	OutputStream   io.Writer
	ErrorStream    io.Writer
}

// New returns a new handler.
func New(opts *Opts) common.Handler {
	releaseDir := opts.ReleaseDir
	if releaseDir == "" {
		releaseDir = "/release"
	}
	inputStream := opts.InputStream
	if inputStream == nil {
		inputStream = os.Stdin
	}
	outputStream := opts.OutputStream
	if outputStream == nil {
		outputStream = os.Stdout
	}
	errorStream := opts.ErrorStream
	if errorStream == nil {
		errorStream = os.Stderr
	}
	return &Handler{
		s3Client:       opts.S3Client,
		dynamoDBClient: opts.DynamoDBClient,
		ecrClient:      opts.ECRClient,
		releaseDir:     releaseDir,
		inputStream:    inputStream,
		outputStream:   outputStream,
		errorStream:    errorStream,
		styles:         initStyles(),
	}
}

func (handler *Handler) getS3Client() s3iface.S3API {
	if handler.s3Client == nil {
		handler.s3Client = s3.New(handler.awsSession)
	}
	return handler.s3Client
}

func (handler *Handler) getDynamoDBClient() dynamodbiface.DynamoDBAPI {
	if handler.dynamoDBClient == nil {
		handler.dynamoDBClient = dynamodb.New(handler.awsSession)
	}
	return handler.dynamoDBClient
}

func (handler *Handler) getECRClient() ecriface.ECRAPI {
	if handler.ecrClient == nil {
		handler.ecrClient = ecr.New(handler.awsSession)
	}
	return handler.ecrClient
}

func randHexPostfix() string {
	randomBytes := make([]byte, 20)
	rand.Read(randomBytes)
	randomBytes = append(randomBytes, big.NewInt(time.Now().UnixNano()).Bytes()...)
	return fmt.Sprintf("%x", sha256.Sum256(randomBytes))[:16]
}

func filterPrefix(input []string, prefix string) []string {
	var result []string
	for _, item := range input {
		if strings.HasPrefix(item, prefix) {
			result = append(result, item)
		}
	}
	return result
}

func (handler *Handler) downloadRelease(request *common.PrepareTerraformRequest) error {
	buffer := &aws.WriteAtBuffer{}
	component, _ := request.Config["component"].(string)
	team, _ := request.Config["team"].(string)
	releaseKey := fmt.Sprintf("%s/%s/%s", team, component, request.Version)
	size, err := s3manager.NewDownloaderWithClient(handler.getS3Client()).Download(buffer, &s3.GetObjectInput{
		Bucket: &handler.releaseBucket,
		Key:    &releaseKey,
	})
	if err != nil {
		return err
	}
	return common.UnzipRelease(bytes.NewReader(buffer.Bytes()), size, handler.releaseDir, component, request.Version)
}

func (handler *Handler) PrepareTerraform(request *common.PrepareTerraformRequest, response *common.PrepareTerraformResponse) error {
	if err := handler.downloadRelease(request); err != nil {
		return err
	}
	return nil
}
