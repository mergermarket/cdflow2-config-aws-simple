![GitHub Workflow Status](https://img.shields.io/github/actions/workflow/status/mergermarket/cdflow2-config-aws-simple/unit-tests.yml?label=unit%20tests)
![GitHub Workflow Status](https://img.shields.io/github/actions/workflow/status/mergermarket/cdflow2-config-aws-simple/publish.yml?label=publish)
![Docker Image Size (latest by date)](https://img.shields.io/docker/image-size/mergermarket/cdflow2-config-aws-simple)

## Datadog monitoring

To enable sending cdflow2 events to Datadog a secret must be added to AWS Secrets manager. 
The secret name must be `cdflow2/datadog/datadog-api-key` and the value is a valid Datadog API key.
