//go:generate go run $GOPATH/src/github.com/mjibson/esc/main.go -o ./resources/RESOURCES.go -pkg resources ./resources/source

package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	awsLambdaCtx "github.com/aws/aws-lambda-go/lambdacontext"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	sparta "github.com/mweagle/Sparta"
	spartaAWS "github.com/mweagle/Sparta/aws"
	spartaCF "github.com/mweagle/Sparta/aws/cloudformation"
	spartaCFResources "github.com/mweagle/Sparta/aws/cloudformation/resources"
	spartaIAM "github.com/mweagle/Sparta/aws/iam"
	"github.com/mweagle/SpartaOmega/resources"
	gocf "github.com/mweagle/go-cloudformation"
	"github.com/sirupsen/logrus"

	"github.com/spf13/cobra"
)

// HTTPServerPort is the port the "HelloWorld" service binds to
const HTTPServerPort = 9999

// SSHKeyName is the SSH KeyName to use when provisioning new EC2 instance
var SSHKeyName string

// Given a slice of EC2 images, return the most recently Created one.
func mostRecentImageID(images []*ec2.Image) (string, error) {
	if len(images) <= 0 {
		return "", fmt.Errorf("No images to search")
	}
	var mostRecentImage *ec2.Image
	for _, eachImage := range images {
		if nil == mostRecentImage {
			mostRecentImage = eachImage
		} else {
			curTime, curTimeErr := time.Parse(time.RFC3339, *mostRecentImage.CreationDate)
			if nil != curTimeErr {
				return "", curTimeErr
			}
			testTime, testTimeErr := time.Parse(time.RFC3339, *eachImage.CreationDate)
			if nil != testTimeErr {
				return "", testTimeErr
			}
			if (testTime.Unix() - curTime.Unix()) > 0 {
				mostRecentImage = eachImage
			}
		}
	}
	return *mostRecentImage.ImageId, nil
}

// Lambda CustomResource function that looks up the latest Ubuntu AMI ID for the
// current region and returns a map with the latest AMI IDs via the resource's
// outputs.  For this example we're going to look for the latest Ubuntu 16.04
// release.
// Ref: https://help.ubuntu.com/community/EC2StartersGuide#Official_Ubuntu_Cloud_Guest_Amazon_Machine_Images_.28AMIs.29
func ubuntuAMICustomResource(ctx context.Context,
	event spartaCFResources.CloudFormationLambdaEvent) (map[string]interface{}, error) {
	logger, _ := ctx.Value(sparta.ContextKeyLogger).(*logrus.Logger)
	lambdaCtx, _ := awsLambdaCtx.FromContext(ctx)

	opResults := make(map[string]interface{})
	var opErr error
	if event.RequestType == "Create" {

		// Setup the common filters
		describeImageFilters := []*ec2.Filter{}
		describeImageFilters = append(describeImageFilters, &ec2.Filter{
			Name:   aws.String("name"),
			Values: []*string{aws.String("*hvm-ssd/ubuntu-xenial-16.04-amd64-server*")},
		})
		describeImageFilters = append(describeImageFilters, &ec2.Filter{
			Name:   aws.String("root-device-type"),
			Values: []*string{aws.String("ebs")},
		})
		describeImageFilters = append(describeImageFilters, &ec2.Filter{
			Name:   aws.String("architecture"),
			Values: []*string{aws.String("x86_64")},
		})
		describeImageFilters = append(describeImageFilters, &ec2.Filter{
			Name:   aws.String("virtualization-type"),
			Values: []*string{aws.String("hvm")},
		})

		// Get the HVM AMIs
		params := &ec2.DescribeImagesInput{
			Filters: describeImageFilters,
			Owners:  []*string{aws.String("099720109477")},
		}
		logger, _ := ctx.Value(sparta.ContextKeyLogger).(*logrus.Logger)
		ec2Svc := ec2.New(spartaAWS.NewSession(logger))
		describeImagesOutput, describeImagesOutputErr := ec2Svc.DescribeImages(params)
		if nil != describeImagesOutputErr {
			return nil, describeImagesOutputErr
		}
		logger.WithFields(logrus.Fields{
			"DescribeImagesOutput":    describeImagesOutput,
			"DescribeImagesOutputErr": describeImagesOutputErr,
		}).Info("Results")

		amiID, amiIDErr := mostRecentImageID(describeImagesOutput.Images)
		if nil != amiIDErr {
			return nil, amiIDErr
		}

		// Set the HVM type
		opResults = map[string]interface{}{
			"HVM": amiID,
		}
		logger.WithFields(logrus.Fields{
			"Outputs": opResults,
		}).Info("CustomResource outputs")
	}

	cfErr := spartaCFResources.SendCloudFormationResponse(lambdaCtx,
		&event,
		opResults,
		opErr,
		logger)
	if opErr == nil {
		opErr = cfErr
	}
	return opResults, opErr
}

// The CloudFormation template decorator that inserts all the other
// AWS components we need to support this deployment...
func lambdaDecorator(customResourceAMILookupName string) sparta.TemplateDecorator {

	return func(serviceName string,
		lambdaResourceName string,
		lambdaResource gocf.LambdaFunction,
		resourceMetadata map[string]interface{},
		S3Bucket string,
		S3Key string,
		buildID string,
		template *gocf.Template,
		context map[string]interface{},
		logger *logrus.Logger) error {

		// Create the launch configuration with Metadata to download the ZIP file, unzip it & launch the
		// golang binary...
		ec2SecurityGroupResourceName := sparta.CloudFormationResourceName("SpartaOmegaSecurityGroup",
			"SpartaOmegaSecurityGroup")
		asgLaunchConfigurationName := sparta.CloudFormationResourceName("SpartaOmegaASGLaunchConfig",
			"SpartaOmegaASGLaunchConfig")
		asgResourceName := sparta.CloudFormationResourceName("SpartaOmegaASG",
			"SpartaOmegaASG")
		ec2InstanceRoleName := sparta.CloudFormationResourceName("SpartaOmegaEC2InstanceRole",
			"SpartaOmegaEC2InstanceRole")
		ec2InstanceProfileName := sparta.CloudFormationResourceName("SpartaOmegaEC2InstanceProfile",
			"SpartaOmegaEC2InstanceProfile")

		//////////////////////////////////////////////////////////////////////////////
		// 1 - Create the security group for the SpartaOmega EC2 instance
		ec2SecurityGroup := &gocf.EC2SecurityGroup{
			GroupDescription: gocf.String("SpartaOmega Security Group"),
			SecurityGroupIngress: &gocf.EC2SecurityGroupIngressPropertyList{
				gocf.EC2SecurityGroupIngressProperty{
					CidrIP:     gocf.String("0.0.0.0/0"),
					IPProtocol: gocf.String("tcp"),
					FromPort:   gocf.Integer(HTTPServerPort),
					ToPort:     gocf.Integer(HTTPServerPort),
				},
				gocf.EC2SecurityGroupIngressProperty{
					CidrIP:     gocf.String("0.0.0.0/0"),
					IPProtocol: gocf.String("tcp"),
					FromPort:   gocf.Integer(22),
					ToPort:     gocf.Integer(22),
				},
			},
		}
		template.AddResource(ec2SecurityGroupResourceName, ec2SecurityGroup)
		//////////////////////////////////////////////////////////////////////////////
		// 2 - Create the ASG and associate the userdata with the EC2 init
		// EC2 Instance Role...
		statements := sparta.CommonIAMStatements.Core

		// Add the statement that allows us to fetch the S3 object with this compiled
		// binary
		statements = append(statements, spartaIAM.PolicyStatement{
			Effect:   "Allow",
			Action:   []string{"s3:GetObject"},
			Resource: gocf.String(fmt.Sprintf("arn:aws:s3:::%s/%s", S3Bucket, S3Key)),
		})
		iamPolicyList := gocf.IAMRolePolicyList{}
		iamPolicyList = append(iamPolicyList,
			gocf.IAMRolePolicy{
				PolicyDocument: sparta.ArbitraryJSONObject{
					"Version":   "2012-10-17",
					"Statement": statements,
				},
				PolicyName: gocf.String("EC2Policy"),
			},
		)
		ec2InstanceRole := &gocf.IAMRole{
			AssumeRolePolicyDocument: sparta.AssumePolicyDocument,
			Policies:                 &iamPolicyList,
		}
		template.AddResource(ec2InstanceRoleName, ec2InstanceRole)

		// Create the instance profile
		ec2InstanceProfile := &gocf.IAMInstanceProfile{
			Path:  gocf.String("/"),
			Roles: gocf.StringList(gocf.Ref(ec2InstanceRoleName)),
		}
		template.AddResource(ec2InstanceProfileName, ec2InstanceProfile)

		//Now setup the properties map, expand the userdata, and attach it...
		userDataProps := map[string]interface{}{
			"S3Bucket":         S3Bucket,
			"S3Key":            S3Key,
			"SpartaBinaryName": sparta.SpartaBinaryName,
			"ServiceName":      serviceName,
		}

		userDataTemplateInput, userDataTemplateInputErr := resources.FSString(false,
			"/resources/source/userdata.sh")
		if nil != userDataTemplateInputErr {
			return userDataTemplateInputErr
		}
		userDataExpression, userDataExpressionErr := spartaCF.ConvertToTemplateExpression(strings.NewReader(userDataTemplateInput), userDataProps)
		if nil != userDataExpressionErr {
			return userDataExpressionErr
		}

		logger.WithFields(logrus.Fields{
			"Parameters": userDataProps,
			"Expanded":   userDataExpression,
		}).Debug("Expanded userdata")

		asgLaunchConfigurationResource := &gocf.AutoScalingLaunchConfiguration{
			ImageID:            gocf.GetAtt(customResourceAMILookupName, "HVM"),
			InstanceType:       gocf.String("t2.micro"),
			KeyName:            gocf.String(SSHKeyName),
			IamInstanceProfile: gocf.Ref(ec2InstanceProfileName).String(),
			UserData:           gocf.Base64(userDataExpression),
			SecurityGroups:     gocf.StringList(gocf.GetAtt(ec2SecurityGroupResourceName, "GroupId")),
		}

		launchConfigResource := template.AddResource(asgLaunchConfigurationName,
			asgLaunchConfigurationResource)
		launchConfigResource.DependsOn = append(launchConfigResource.DependsOn,
			customResourceAMILookupName)

		// Create the ASG
		asgResource := &gocf.AutoScalingAutoScalingGroup{
			// Empty Region is equivalent to all region AZs
			// Ref: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/intrinsic-function-reference-getavailabilityzones.html
			AvailabilityZones:       gocf.GetAZs(gocf.String("")),
			LaunchConfigurationName: gocf.Ref(asgLaunchConfigurationName).String(),
			MaxSize:                 gocf.String("1"),
			MinSize:                 gocf.String("1"),
		}
		template.AddResource(asgResourceName, asgResource)
		return nil
	}
}

func helloWorldLambda() (string, error) {
	return "Hello world from SpartaOmega!", nil
}

func helloWorldResource(w http.ResponseWriter, r *http.Request) {

	r.Header.Set("Content-Type", "text/plain")
	helloData, _ := helloWorldLambda()
	fmt.Fprintf(w, helloData)
}

////////////////////////////////////////////////////////////////////////////////
// Main
func main() {

	// Custom command to startup a simple HelloWorld HTTP server
	httpServerCommand := &cobra.Command{
		Use:   "httpServer",
		Short: "Sample HelloWorld HTTP server",
		Long:  fmt.Sprintf("Sample HelloWorld HTTP server that binds to port: %d", HTTPServerPort),
		RunE: func(cmd *cobra.Command, args []string) error {
			http.HandleFunc("/", helloWorldResource)
			return http.ListenAndServe(fmt.Sprintf(":%d", HTTPServerPort), nil)
		},
	}
	sparta.CommandLineOptions.Root.AddCommand(httpServerCommand)

	// And add the SSHKeyName option to the provision step
	sparta.CommandLineOptions.Provision.Flags().StringVarP(&SSHKeyName,
		"key",
		"k",
		"",
		"SSH Key Name to use for EC2 instances")

	// The primary lambda function
	lambdaFn := sparta.HandleAWSLambda(sparta.LambdaName(helloWorldLambda),
		helloWorldLambda,
		sparta.IAMRoleDefinition{})

	// Lambda custom resource to lookup the latest Ubuntu AMIs
	iamRoleCustomResource := sparta.IAMRoleDefinition{}
	iamRoleCustomResource.Privileges = append(iamRoleCustomResource.Privileges,
		sparta.IAMRolePrivilege{
			Actions:  []string{"ec2:DescribeImages"},
			Resource: "*",
		})

	customResourceLambdaOptions := sparta.LambdaFunctionOptions{
		MemorySize: 128,
		Timeout:    30,
	}
	amiIDCustomResourceName, _ := lambdaFn.RequireCustomResource(iamRoleCustomResource,
		ubuntuAMICustomResource,
		&customResourceLambdaOptions,
		nil)

	defaultDecorator := lambdaDecorator(amiIDCustomResourceName)
	lambdaFn.Decorators = []sparta.TemplateDecoratorHandler{
		sparta.TemplateDecoratorHookFunc(defaultDecorator),
	}

	var lambdaFunctions []*sparta.LambdaAWSInfo
	lambdaFunctions = append(lambdaFunctions, lambdaFn)

	stackName := spartaCF.UserScopedStackName("SpartaOmega")
	err := sparta.Main(stackName,
		fmt.Sprintf("Provision AWS Lambda and EC2 instance with same code"),
		lambdaFunctions,
		nil,
		nil)
	if err != nil {
		os.Exit(1)
	}
}
