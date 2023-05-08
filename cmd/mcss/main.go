package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

type MyEvent struct {
	RequestType string `json:"requestType"`
}

type Response struct {
	Status int    `json:"statusCode"`
	Body   string `json:"body"`
}

type ResBody struct {
	Message   string `json:"message"`
	IPAddress string `json:"ipAddress,omitempty"`
}

func handleRequest(ctx context.Context, request MyEvent) (Response, error) {
	ec2Handler := NewEC2Handler()

	switch request.RequestType {
	case "start":
		status := 202
		r, _ := ec2Handler.StartInstance(ctx)
		rb := ResBody{Message: r.Status}
		if r.Status == "running" {
			rb.IPAddress = r.IPAddress
			status = 200
		}
		out, _ := json.Marshal(rb)
		return Response{Status: status, Body: string(out)}, nil
	case "stop":
		status := 202
		r, _ := ec2Handler.StopInstance(ctx)
		rb := ResBody{Message: r.Status}
		if r.Status == "stopped" {
			rb.Message = r.Status
			status = 200
		}
		out, _ := json.Marshal(ResBody{Message: r.Status})
		return Response{Status: status, Body: string(out)}, nil
	case "status":
		status, _ := ec2Handler.InstanceStatus(ctx)
		m, _ := json.Marshal(status)
		return Response{Status: 200, Body: string(m)}, nil
	default:
		out, _ := json.Marshal(request)
		return Response{Status: 500, Body: string(out)}, nil
	}
}

func main() {
	lambda.Start(handleRequest)
}

type EC2Status struct {
	InstanceId string `json:"instanceId"`
	Status     string `json:"status"`
	IPAddress  string `json:"ipAddress,omitempty"`
}

type EC2Interface interface {
	StartInstance(ctx context.Context) (*EC2Status, error)
	StopInstance(ctx context.Context) (*EC2Status, error)
	InstanceStatus(ctx context.Context) (*EC2Status, error)
}

type EC2Handler struct {
}

func NewEC2Handler() EC2Interface {
	return &EC2Handler{}
}

func (ec2Handler *EC2Handler) StartInstance(ctx context.Context) (*EC2Status, error) {
	instanceId := os.Getenv("INSTANCE_ID")
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		fmt.Println(err)
		return &EC2Status{}, err
	}
	ec2Client := ec2.NewFromConfig(cfg)
	b := true
	input := &ec2.DescribeInstanceStatusInput{
		InstanceIds:         []string{instanceId},
		IncludeAllInstances: &b,
	}
	output, err := ec2Client.DescribeInstanceStatus(ctx, input)
	if err != nil {
		fmt.Println("INST ERR", err)
		return &EC2Status{}, err
	}

	result, err2 := ec2Client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
		InstanceIds: []string{instanceId},
	})
	if err2 != nil {
		fmt.Println("INST 22222", err)
		return &EC2Status{}, err
	}

	isRunning := false
	var instance types.InstanceStatus

	for _, instanceStatus := range output.InstanceStatuses {
		fmt.Printf("%s: %s\n", *instanceStatus.InstanceId, instanceStatus.InstanceState.Name)
		instance = instanceStatus
		if *instanceStatus.InstanceId == instanceId && instanceStatus.InstanceState.Name == "running" {
			isRunning = true
		}
	}

	if !isRunning {
		runInstance := &ec2.StartInstancesInput{
			InstanceIds: []string{instanceId},
		}
		fmt.Printf("Start %s", instanceId)
		if outputStart, errInstance := ec2Client.StartInstances(ctx, runInstance); errInstance != nil {
			return &EC2Status{
				InstanceId: instanceId,
				Status:     "pending",
			}, errInstance
		} else {
			fmt.Println(outputStart.StartingInstances)
			return &EC2Status{
				InstanceId: *instance.InstanceId,
				Status:     string(instance.InstanceState.Name),
			}, nil
		}
	} else {
		fmt.Printf("Finally Started, address: %s \n", "before")
		var address string
		for _, r := range result.Reservations {
			fmt.Println("Reservation ID: " + *r.ReservationId)
			fmt.Println("Instance IDs:")
			for _, i := range r.Instances {
				address = *i.PublicIpAddress
			}
		}

		return &EC2Status{
			InstanceId: *instance.InstanceId,
			Status:     string(instance.InstanceState.Name),
			IPAddress:  address,
		}, nil
	}
}

func (ec2Handler *EC2Handler) StopInstance(ctx context.Context) (*EC2Status, error) {
	instanceId := os.Getenv("INSTANCE_ID")
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		fmt.Println(err)
		return &EC2Status{}, err
	}
	ec2Client := ec2.NewFromConfig(cfg)
	b := true
	input := &ec2.DescribeInstanceStatusInput{
		InstanceIds:         []string{instanceId},
		IncludeAllInstances: &b,
	}
	output, err := ec2Client.DescribeInstanceStatus(ctx, input)
	if err != nil {
		fmt.Println(err)
		return &EC2Status{}, err
	}

	for _, instanceStatus := range output.InstanceStatuses {
		fmt.Printf("%s: %s\n", *instanceStatus.InstanceId, instanceStatus.InstanceState.Name)

		if *instanceStatus.InstanceId == instanceId && instanceStatus.InstanceState.Name == "running" {
			stopInstance := &ec2.StopInstancesInput{
				InstanceIds: []string{instanceId},
			}
			if _, errInstance := ec2Client.StopInstances(ctx, stopInstance); errInstance != nil {
				return &EC2Status{
					InstanceId: instanceId,
					Status:     "stopping",
				}, nil
			} else {
				return &EC2Status{
					InstanceId: instanceId,
					Status:     "Cant stop??",
				}, nil
			}
		} else if *instanceStatus.InstanceId == instanceId && (instanceStatus.InstanceState.Name == "stopping" || instanceStatus.InstanceState.Name == "shutting-down") {
			fmt.Println("Stopping second yo")
			return &EC2Status{
				InstanceId: instanceId,
				Status:     "stopping",
			}, nil
		} else if *instanceStatus.InstanceId == instanceId && (instanceStatus.InstanceState.Name == "stopped" || instanceStatus.InstanceState.Name == "terminated") {
			fmt.Println("Stopped yo")
			return &EC2Status{
				InstanceId: instanceId,
				Status:     "stopped",
			}, nil
		}
	}

	return &EC2Status{
		InstanceId: instanceId,
		Status:     "stopping",
	}, errors.New("Unknown problem occurred. Server state unknown.")
}

func (ec2Handler *EC2Handler) InstanceStatus(ctx context.Context) (statuses *EC2Status, err error) {
	instanceId := os.Getenv("INSTANCE_ID")
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		fmt.Println(err)
		return &EC2Status{}, err
	}
	ec2Client := ec2.NewFromConfig(cfg)
	b := true
	input := &ec2.DescribeInstanceStatusInput{
		InstanceIds:         []string{instanceId},
		IncludeAllInstances: &b,
	}
	output, err := ec2Client.DescribeInstanceStatus(ctx, input)
	if err != nil {
		fmt.Println(err)
		return
	}
	var status EC2Status
	for _, statusData := range output.InstanceStatuses {
		status = EC2Status{
			InstanceId: *statusData.InstanceId,
			Status:     string(statusData.InstanceState.Name),
		}
	}
	return &status, nil
}
