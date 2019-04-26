package main

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"

	circleci "github.com/jszwedko/go-circleci"
)

type errorMsg struct {
	Error string `json:"error"`
}

func handleRequest(req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	circleCIToken := os.Getenv("CIRCLECI_TOKEN")
	if circleCIToken == "" {
		return serverError(errors.New("Missing Circle CI Token"))
	}

	serverToken := os.Getenv("TOKEN")
	if serverToken == "" {
		return serverError(errors.New("Missing Token to validate the request"))
	}

	token := req.QueryStringParameters["token"]
	if token == "" || token != serverToken {
		msg := errorMsg{
			Error: "missing or invalid token",
		}
		return clientError(http.StatusBadRequest, msg)
	}

	vcsType := req.QueryStringParameters["vcs-type"]
	if vcsType == "" {
		msg := errorMsg{
			Error: "missing vcs-type",
		}
		return clientError(http.StatusBadRequest, msg)
	}

	username := req.QueryStringParameters["username"]
	if username == "" {
		msg := errorMsg{
			Error: "missing username",
		}
		return clientError(http.StatusBadRequest, msg)
	}

	project := req.QueryStringParameters["project"]
	if project == "" {
		msg := errorMsg{
			Error: "missing project name",
		}
		return clientError(http.StatusBadRequest, msg)
	}

	buildNum := req.QueryStringParameters["build_num"]
	if buildNum == "" {
		msg := errorMsg{
			Error: "missing build number",
		}
		return clientError(http.StatusBadRequest, msg)
	}

	s3Bucket := req.QueryStringParameters["bucket"]
	if buildNum == "" {
		msg := errorMsg{
			Error: "missing bucket name",
		}
		return clientError(http.StatusBadRequest, msg)
	}

	artifactList, err := getCircleCIArtifacat(circleCIToken, vcsType, username, project, buildNum)
	if err != nil {
		return serverError(err)
	}

	for _, artifact := range artifactList {
		artifactPath := strings.Split(artifact.PrettyPath, "/")
		fileName := artifactPath[len(artifactPath)-1]
		err := downloadFile(fileName, artifact.URL)
		if err != nil {
			return serverError(err)
		}
		s, err := session.NewSession(&aws.Config{Region: aws.String("us-east-1")})
		if err != nil {
			return serverError(err)
		}

		err = uploadFile(s, fileName, s3Bucket)
		if err != nil {
			return serverError(err)
		}

	}

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Body:       string("{\"status\":\"ok\"}"),
	}, nil
}

func clientError(status int, message errorMsg) (events.APIGatewayProxyResponse, error) {
	jsonString, _ := json.Marshal(message)
	return events.APIGatewayProxyResponse{
		StatusCode: status,
		Body:       string(jsonString),
	}, nil
}

func serverError(err error) (events.APIGatewayProxyResponse, error) {
	msg := errorMsg{
		Error: err.Error(),
	}
	jsonString, _ := json.Marshal(msg)
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusInternalServerError,
		Body:       string(jsonString),
	}, nil
}

func getCircleCIArtifacat(circleCIToken, vcsType, username, project, buildNum string) ([]*circleci.Artifact, error) {
	circleCIClient := &circleci.Client{Token: circleCIToken}

	buildNumber, _ := strconv.Atoi(buildNum)
	artifactsList, err := circleCIClient.ListBuildArtifacts(username, project, buildNumber)
	if err != nil {
		return nil, err
	}
	return artifactsList, nil
}

// DownloadFile download the file from a specific URL
func downloadFile(fileName string, url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(filepath.Join("/tmp", fileName))
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

// UploadFile upload the file to S3
func uploadFile(awsSession *session.Session, filename string, bucket string) error {
	s3Uploader := s3manager.NewUploader(awsSession)
	reader, err := os.Open(filepath.Join("/tmp", filename))
	if err != nil {
		return err
	}
	defer reader.Close()

	uploadInput := &s3manager.UploadInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(filename),
		Body:   reader,
		CacheControl: aws.String("no-cache"),
		ContentType: aws.String("application/x-gzip"),
		ACL: aws.String("public-read"),
	}
	_, err = s3Uploader.Upload(uploadInput)
	if err != nil {
		return err
	}
	return nil
}

func main() {
	lambda.Start(handleRequest)
}
