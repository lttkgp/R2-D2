package main

import (
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	as "github.com/aws/aws-sdk-go/service/applicationautoscaling"
	"go.uber.org/zap"
)

// NewScalingConfig initializes ScalingConfig with default values for Min = 1, Max = 5 capacity and TargetValue = 70.00 for
// capacity utilization
// AWS Docs:
// https://docs.aws.amazon.com/amazondynamodb/latest/developerguide/AutoScaling.html
// https://docs.aws.amazon.com/amazondynamodb/latest/developerguide/AutoScaling.CLI.html#AutoScaling.CLI.RegisterScalableTarget
func NewScalingConfig(awsSession *session.Session, logger *zap.Logger) *ScalingConfig {
	svc := as.New(awsSession)
	return &ScalingConfig{
		svc:    svc,
		logger: logger,
		Table: Table{
			Name:    tableName,
			Indexes: listOfIndexes,
		},
		Min:         1,
		Max:         5,
		TargetValue: 70.00,
	}
}

// ScalingConfig contains autoscaling configuration
type ScalingConfig struct {
	svc    *as.ApplicationAutoScaling
	logger *zap.Logger
	Table  Table

	// Min can be as low as 1 and must not be higher than Max
	Min int64

	// Max can be as high as 10,000 and must not be lower than Min
	Max int64

	// TargetValue must be between '10.0' and '90.0'
	TargetValue float64
}

// Table has the Name of the DynamoDB table and list of Indexes
type Table struct {
	Name    string
	Indexes []string
}

// SetAutoScaling will check if application scaling targets for provided DynamoDB table and it's indexes.
// If the targets don't exist it will create new ones. After making sure targets exists it will create/update
// scaling policies using Target Tracking policy. As a result this function enforces table and index autoscaling.
func (sc *ScalingConfig) SetAutoScaling() {
	sc.checkTargetTable()
	sc.checkTargetTableIndexes()
}

func (sc *ScalingConfig) checkTargetTable() {
	scalableDimensions := []string{
		as.ScalableDimensionDynamodbTableWriteCapacityUnits,
		as.ScalableDimensionDynamodbTableReadCapacityUnits,
	}

	resourceID := "table/" + sc.Table.Name
	for _, scalableDimension := range scalableDimensions {
		result, err := sc.svc.DescribeScalableTargets(&as.DescribeScalableTargetsInput{
			ServiceNamespace:  aws.String(as.ServiceNamespaceDynamodb),
			ResourceIds:       aws.StringSlice([]string{resourceID}),
			ScalableDimension: aws.String(scalableDimension),
		})
		if err != nil {
			sc.logger.Error("Error describing scalable targets for DynamoDB", zap.Error(err))
			return
		}

		if len(result.ScalableTargets) == 0 {
			err := sc.registerScalingTarget(scalableDimension, resourceID)
			if err != nil {
				sc.logger.Error("Error registering scalable targets for DynamoDB", zap.Error(err))
			}
		}

		err = sc.putScalingPolicy(scalableDimension, resourceID)
		if err != nil {
			sc.logger.Error("Error putting scaling policy", zap.Error(err))
		}
	}
}

func (sc *ScalingConfig) checkTargetTableIndexes() {
	scalableDimensions := []string{
		as.ScalableDimensionDynamodbIndexReadCapacityUnits,
		as.ScalableDimensionDynamodbIndexWriteCapacityUnits,
	}
	for _, index := range sc.Table.Indexes {
		resourceID := "table/" + sc.Table.Name + "/index/" + index
		for _, scalableDimension := range scalableDimensions {
			result, err := sc.svc.DescribeScalableTargets(&as.DescribeScalableTargetsInput{
				ServiceNamespace:  aws.String(as.ServiceNamespaceDynamodb),
				ResourceIds:       aws.StringSlice([]string{resourceID}),
				ScalableDimension: aws.String(scalableDimension),
			})
			if err != nil {
				sc.logger.Error("Error describing scalable targets for DynamoDB", zap.Error(err))
				return
			}

			if len(result.ScalableTargets) == 0 {
				err := sc.registerScalingTarget(scalableDimension, resourceID)
				if err != nil {
					sc.logger.Error("Error registering scalable targets for DynamoDB", zap.Error(err))
				}
			}

			err = sc.putScalingPolicy(scalableDimension, resourceID)
			if err != nil {
				sc.logger.Error("Error putting scaling policy", zap.Error(err))
			}
		}
	}
}

func (sc *ScalingConfig) registerScalingTarget(scalableDimension, resourceID string) error {
	_, err := sc.svc.RegisterScalableTarget(&as.RegisterScalableTargetInput{
		MaxCapacity:       aws.Int64(sc.Max),
		MinCapacity:       aws.Int64(sc.Min),
		ResourceId:        aws.String(resourceID),
		ScalableDimension: aws.String(scalableDimension),
		ServiceNamespace:  aws.String(as.ServiceNamespaceDynamodb),
	})
	if err != nil {
		return err
	}

	return nil
}

func (sc *ScalingConfig) putScalingPolicy(scalableDimension, resourceID string) error {
	var metricType string
	if strings.Contains(scalableDimension, "Read") {
		metricType = as.MetricTypeDynamoDbreadCapacityUtilization
	} else {
		metricType = as.MetricTypeDynamoDbwriteCapacityUtilization
	}

	_, err := sc.svc.PutScalingPolicy(&as.PutScalingPolicyInput{
		PolicyName:        aws.String("ScalingPolicy"),
		PolicyType:        aws.String(as.PolicyTypeTargetTrackingScaling),
		ResourceId:        aws.String(resourceID),
		ScalableDimension: aws.String(scalableDimension),
		ServiceNamespace:  aws.String(as.ServiceNamespaceDynamodb),
		TargetTrackingScalingPolicyConfiguration: &as.TargetTrackingScalingPolicyConfiguration{
			PredefinedMetricSpecification: &as.PredefinedMetricSpecification{
				PredefinedMetricType: aws.String(metricType),
			},
			TargetValue: aws.Float64(sc.TargetValue),
		},
	})
	if err != nil {
		return err
	}

	return nil
}
