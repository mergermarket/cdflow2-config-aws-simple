package handler

import (
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"math/big"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/aws/aws-sdk-go/service/ecr/ecriface"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
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
	ReleaseFolder  string
	releaseBucket  string
	tfstateBucket  string
	tflocksTable   string
	lambdaBucket   string
	InputStream    io.Reader
	OutputStream   io.Writer
	ErrorStream    io.Writer
	ReleaseLoader  common.ReleaseLoader
	ReleaseSaver   common.ReleaseSaver
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
	ReleaseSaver   common.ReleaseSaver
	ReleaseLoader  common.ReleaseLoader
}

// New returns a new handler.
func New(opts *Opts) *Handler {
	releaseDir := opts.ReleaseDir
	if releaseDir == "" {
		releaseDir = "/release"
	}
	InputStream := opts.InputStream
	if InputStream == nil {
		InputStream = os.Stdin
	}
	OutputStream := opts.OutputStream
	if OutputStream == nil {
		OutputStream = os.Stdout
	}
	ErrorStream := opts.ErrorStream
	if ErrorStream == nil {
		ErrorStream = os.Stderr
	}

	return &Handler{
		s3Client:       opts.S3Client,
		dynamoDBClient: opts.DynamoDBClient,
		ecrClient:      opts.ECRClient,
		ReleaseFolder:  releaseDir,
		InputStream:    InputStream,
		OutputStream:   OutputStream,
		ErrorStream:    ErrorStream,
		ReleaseSaver:   common.CreateReleaseSaver(),
		ReleaseLoader:  common.CreateReleaseLoader(),
		styles:         initStyles(),
	}
}

func (handler *Handler) getS3Client() s3iface.S3API {
	if handler.s3Client == nil {
		if handler.awsSession == nil {
			log.Panic("No AWS session")
		}
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

func releaseS3Key(team, component, version string) string {
	return fmt.Sprintf("%s/%s/%s-%s.zip", team, component, component, version)
}

func (h *Handler) getTeam(team interface{}) (string, error) {
	teamString, ok := team.(string)
	if !ok || teamString == "" {
		return "", fmt.Errorf("cdflow.yaml error: config.params.team must be set to a non-empty string")
	}
	return teamString, nil
}
