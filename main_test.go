package main

import (
	"bytes"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
	"github.com/golang/mock/gomock"
	"testing"
)

func TestExecuteWithParameters(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockAwsSsm := NewMockssmClient(ctrl)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	mockAwsSsm.EXPECT().GetParameter(gomock.Any(), &ssm.GetParameterInput{
		Name:           aws.String("test1SsmParameter"),
		WithDecryption: true,
	}).Return(&ssm.GetParameterOutput{
		Parameter: &types.Parameter{
			Value: aws.String("test1"),
		},
	}, nil)

	err := (&handler{
		osEnviron: []string{
			"TEST1_SSM_PARAMETER_NAME=test1SsmParameter",
			"TEST2=test2",
		},
		osArgs: []string{
			"lambda-docker-wrapper",
			"bash", "-c", "echo ${TEST1} ${TEST2}",
		},
		stdout: &stdout,
		stderr: &stderr,

		awsSsm: mockAwsSsm,
	}).handle()
	if err != nil {
		t.Fatal(err)
	}

	output := stdout.String()
	expected := "test1 test2\n"

	if output != expected {
		t.Errorf("Expected output: %v, got: %v", expected, output)
	}
}

func TestFailingCommand(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockAwsSsm := NewMockssmClient(ctrl)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := (&handler{
		osEnviron: []string{},
		osArgs: []string{
			"lambda-docker-wrapper",
			"bash", "-c", "exit 2",
		},
		stdout: &stdout,
		stderr: &stderr,

		awsSsm: mockAwsSsm,
	}).handle()

	if err == nil {
		t.Errorf("Error expected")
	}

	errorMsg := err.Error()
	expected := "external command execution failed: execute external command bash failed with exit code 2: exit status 2"

	if errorMsg != expected {
		t.Errorf("Expected error message: %v, got: %v", expected, errorMsg)
	}
}

func TestErrorGettingSsmParameter(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockAwsSsm := NewMockssmClient(ctrl)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	mockAwsSsm.EXPECT().GetParameter(gomock.Any(), gomock.Any()).Return(
		nil, fmt.Errorf("error in AWS"),
	)

	err := (&handler{
		osEnviron: []string{
			"TEST1_SSM_PARAMETER_NAME=test1SsmParameter",
		},
		osArgs: []string{},
		stdout: &stdout,
		stderr: &stderr,

		awsSsm: mockAwsSsm,
	}).handle()

	if err == nil {
		t.Errorf("Error expected")
	}

	errorMsg := err.Error()
	expected := "cannot get paramter value: cannot get ssm parameter: error in AWS"

	if errorMsg != expected {
		t.Errorf("Expected error message: %v, got: %v", expected, errorMsg)
	}
}
