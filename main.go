package main

import (
	"context"
	"errors"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	gerrors "github.com/pkg/errors"
	"io"
	"os"
	"os/exec"
	"strings"
)

const ssmParameterEnvVarSuffix = "_SSM_PARAMETER_NAME"

func main() {
	lambda.Start(func() error {
		awsSsm, err := newSsmClient()
		if err != nil {
			return gerrors.Wrapf(err, "cannot create ssm client")
		}

		return (&handler{
			osEnviron: os.Environ(),
			osArgs:    os.Args,
			stdout:    os.Stdout,
			stderr:    os.Stderr,

			awsSsm: awsSsm,
		}).handle()
	})
}

type handler struct {
	osEnviron []string
	osArgs    []string
	stdout    io.Writer
	stderr    io.Writer

	awsSsm ssmClient
}

func (h *handler) handle() error {
	parametersFromEnv := h.getEnvVarsWithSuffix(ssmParameterEnvVarSuffix)

	extraEnv := make(map[string]string, len(parametersFromEnv))

	for key, value := range parametersFromEnv {
		secretValue, err := h.getSsmParameterValue(value)
		if err != nil {
			return gerrors.Wrapf(err, "cannot get paramter value")
		}

		secretEnvVarName := strings.TrimSuffix(key, ssmParameterEnvVarSuffix)

		extraEnv[secretEnvVarName] = secretValue
	}

	err := h.executeExternal(
		extraEnv,
		h.osArgs[1],
		h.osArgs[2:]...,
	)
	if err != nil {
		return gerrors.Wrapf(err, "external command execution failed")
	}
	return nil
}

func (h *handler) getEnvVarsWithSuffix(envVarSuffix string) map[string]string {
	result := make(map[string]string)

	for _, envVar := range h.osEnviron {
		pair := strings.SplitN(envVar, "=", 2)
		key := pair[0]
		value := pair[1]

		if strings.HasSuffix(key, envVarSuffix) {
			result[key] = value
		}
	}

	return result
}

func (h *handler) getSsmParameterValue(parameterName string) (string, error) {
	getParameterOutput, err := h.awsSsm.GetParameter(context.TODO(), &ssm.GetParameterInput{
		Name:           aws.String(parameterName),
		WithDecryption: true,
	})
	if err != nil {
		return "", gerrors.Wrapf(err, "cannot get ssm parameter")
	}

	return *getParameterOutput.Parameter.Value, nil
}

func (h *handler) executeExternal(envVars map[string]string, command string, args ...string) error {
	cmd := exec.Command(command, args...)

	cmd.Stdout = h.stdout
	cmd.Stderr = h.stderr

	cmd.Env = append(h.osEnviron, convertEnvVars(envVars)...)

	err := cmd.Run()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			exitCode := exitErr.ExitCode()
			return gerrors.Wrapf(err, "execute external command %v failed with exit code %v", command, exitCode)
		} else {
			return gerrors.Wrapf(err, "external command execution failed %v", command)
		}
	}
	return nil
}

func convertEnvVars(envVars map[string]string) []string {
	result := make([]string, 0, len(envVars))
	for k, v := range envVars {
		envVar := k + "=" + v
		result = append(result, envVar)
	}
	return result
}

func newSsmClient() (ssmClient, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return nil, gerrors.Wrapf(err, "cannot load aws config")
	}

	client := ssm.NewFromConfig(cfg)

	return client, nil
}

type ssmClient interface {
	GetParameter(ctx context.Context, params *ssm.GetParameterInput, optFns ...func(*ssm.Options)) (*ssm.GetParameterOutput, error)
}
