.DEFAULT_GOAL=build
.PHONY: build test get run

clean:
	go clean .

generate: 
	go generate -x
	@echo "Generate complete: `date`"

vet: generate
	go tool vet -composites=false *.go
	
get: clean
	rm -rf $(GOPATH)/src/github.com/aws/aws-sdk-go
	git clone --depth=1 https://github.com/aws/aws-sdk-go $(GOPATH)/src/github.com/aws/aws-sdk-go

	rm -rf $(GOPATH)/src/github.com/go-ini/ini
	git clone --depth=1 https://github.com/go-ini/ini $(GOPATH)/src/github.com/go-ini/ini

	rm -rf $(GOPATH)/src/github.com/jmespath/go-jmespath
	git clone --depth=1 https://github.com/jmespath/go-jmespath $(GOPATH)/src/github.com/jmespath/go-jmespath

	rm -rf $(GOPATH)/src/github.com/Sirupsen/logrus
	git clone --depth=1 https://github.com/Sirupsen/logrus $(GOPATH)/src/github.com/Sirupsen/logrus

	rm -rf $(GOPATH)/src/github.com/mjibson/esc
	git clone --depth=1 https://github.com/mjibson/esc $(GOPATH)/src/github.com/mjibson/esc

	rm -rf $(GOPATH)/src/github.com/crewjam/go-cloudformation
	git clone --depth=1 https://github.com/crewjam/go-cloudformation $(GOPATH)/src/github.com/crewjam/go-cloudformation

	rm -rf $(GOPATH)/src/github.com/spf13/cobra
	git clone --depth=1 https://github.com/spf13/cobra $(GOPATH)/src/github.com/spf13/cobra

	rm -rf $(GOPATH)/src/github.com/spf13/pflag
	git clone --depth=1 https://github.com/spf13/pflag $(GOPATH)/src/github.com/spf13/pflag

	rm -rf $(GOPATH)/src/github.com/asaskevich/govalidator
	git clone --depth=1 https://github.com/asaskevich/govalidator $(GOPATH)/src/github.com/asaskevich/govalidator
	
build: get generate vet
	go build .

test: 
	go test ./test/...

delete:
	go run main.go delete
	
explore:
	go run main.go --level debug explore

provision: generate vet
	go run main.go --level info provision --s3Bucket $(S3_BUCKET) --key sparta-test

describe: generate vet
	clear
	S3_TEST_BUCKET="" SNS_TEST_TOPIC="" DYNAMO_TEST_STREAM="" go run main.go --level info describe --out ./graph.html 

