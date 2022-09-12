# Lambda container image wrapper

## How it works

- Fetches SSM secure string parameters specified in env variables with `_SSM_PARAMETER_NAME` suffix
  (e.g. `DB_PASSWORD_SSM_PARAMETER_NAME`).
- Executes any executable provided as the first argument passing all the other arguments as-is.

## Why we built it?

We wanted to run any executable in on-demand executed Lambdas. Our main use case is running SQL scripts in the same VPC
as RDS instances are residing. However, there are two problems:

- You cannot bind secrets as environment variables. You have to obtain them in the Lambda code. We wanted our images to
  be as runtime agnostic as possible and get secrets from environment variables. This wrapper is the only thing that has
  to be installed to meet our requirements.
- Running a program which will exit (even with the 0 exit code) will fail with an error because it is not what lambda
  expects. You need a long-running program which will handle multiple Lambda requests to avoid this error.

## Limitations

You can run this code as a wrapper for a Lambda executed from AWS API, AWS Console or EventBridge cron, but you
probably **shouldn't use it in Lambdas executed via HTTP endpoint or API gateway**. It can work, but we haven't tested
it, and we expect some problems because our wrapper is spawning a new process using `exec.Run` function.

## Usage

Download the binary in your Dockerfile (Set the LAMBDA_WRAPPER_SHA256 and LAMBDA_WRAPPER_VERSION variables
appropriately):

```dockerfile
ENV LAMBDA_WRAPPER_SHA256=qwertyu123qeqeasdasdae1231dasdasfsadfa1231231dasdasdadasda123131 \
    LAMBDA_WRAPPER_VERSION=0.1.0
RUN curl -L https://github.com/devopsbox-io/lambda-docker-wrapper/releases/download/v${LAMBDA_WRAPPER_VERSION}/lambda-docker-wrapper-${LAMBDA_WRAPPER_VERSION}-linux-amd64 \
        -o /usr/local/bin/lambda-docker-wrapper && \
    echo "${LAMBDA_WRAPPER_SHA256} /usr/local/bin/lambda-docker-wrapper" | sha256sum --check && \
    chmod +x /usr/local/bin/lambda-docker-wrapper
```

Set `lambda-docker-wrapper` as your `ENTRYPOINT` and your executable (e.g. `initialize-mysql.sh`) as `CMD` in the
Lambda `Container image overrides` settings.

If you want to get secure string SSM parameter as an environment variable add `_SSM_PARAMETER_NAME` suffix to the
expected environment variable name. For example `DB_PASSWORD_SSM_PARAMETER_NAME=/app/db-password` will fetch
the `/app/db-password` SSM parameter and store it as the DB_PASSWORD environment variable before executing `CMD`.
