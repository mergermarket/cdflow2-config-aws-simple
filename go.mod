module github.com/mergermarket/cdflow2-config-simple-aws

go 1.13

require (
	github.com/aws/aws-sdk-go v1.38.13
	github.com/logrusorgru/aurora v0.0.0-20200102142835-e9ef32dff381
	github.com/mergermarket/cdflow2-config-common v0.45.0
)

replace github.com/mergermarket/cdflow2-config-common => github.com/mergermarket/cdflow2-config-common v0.45.1-0.20221201115717-80d5ca9df35d
