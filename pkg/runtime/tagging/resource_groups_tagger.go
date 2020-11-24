package tagging

import (
	ackv1alpha1 "github.com/aws/aws-controllers-k8s/apis/core/v1alpha1"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	rgtapi "github.com/aws/aws-sdk-go/service/resourcegroupstaggingapi"
	rgtapiiface "github.com/aws/aws-sdk-go/service/resourcegroupstaggingapi/resourcegroupstaggingapiiface"

	"strings"
)

const (
	defaultAWSTagKey   = "services.k8s.aws/managed-by"
	defaultAWSTagValue = "aws-controllers-k8s"
)

type ResourceGroupsTagger struct {
	rgtapi         rgtapiiface.ResourceGroupsTaggingAPIAPI
	desiredAWSTags map[string]string
}

func (a *ResourceGroupsTagger) SyncTags(ARN *ackv1alpha1.AWSResourceName, session *session.Session) error {
	a.rgtapi = rgtapi.New(session)

	// TODO(zacharya): Currently the Resource Groups Tagging API does not support retrieving
	// tags for an ARN - instead it crawls all the resources of the same type.  Until this
	// is changed we will only try to apply/update the desired tags to avoid AWS API rate issues
	//

	arnString := string(*ARN)
	if err := a.tagResource(arnString); err != nil {
		return err
	}

	return nil
}

func (a *ResourceGroupsTagger) getCurrentResourceTags(ARN string) (map[string]string, error) {
	managedResources, err := a.rgtapi.GetResources(&rgtapi.GetResourcesInput{
		TagFilters: []*rgtapi.TagFilter{
			{
				Key: aws.String(defaultAWSTagKey),
				Values: []*string{
					aws.String(a.desiredAWSTags[defaultAWSTagKey]),
				},
			},
		},
		ResourceTypeFilters: []*string{
			aws.String(getServiceFromARN(ARN)),
		},
	})
	if err != nil {
		return nil, err
	}

	tags := make(map[string]string)

	for _, resource := range managedResources.ResourceTagMappingList {
		if resource.ResourceARN == aws.String(ARN) {
			for _, tag := range resource.Tags {
				tags[*tag.Key] = *tag.Value
			}
		}
	}
	return tags, nil
}

func (a *ResourceGroupsTagger) tagResource(ARN string) error {
	awsTagMap := convertStringPointerMap(a.desiredAWSTags)

	tagResourcesInput := &rgtapi.TagResourcesInput{
		ResourceARNList: []*string{
			aws.String(ARN),
		},
		Tags: awsTagMap,
	}

	if _, err := a.rgtapi.TagResources(tagResourcesInput); err != nil {
		return err
	}
	return nil
}

func (a *ResourceGroupsTagger) untagResource(ARN string, tagKeys []string) error {
	awsTagKeys := convertStringPointerSlice(tagKeys)

	untagResourcesInput := &rgtapi.UntagResourcesInput{
		ResourceARNList: []*string{
			aws.String(ARN),
		},
		TagKeys: awsTagKeys,
	}

	if _, err := a.rgtapi.UntagResources(untagResourcesInput); err != nil {
		return err
	}
	return nil
}

func convertStringPointerMap(m map[string]string) map[string]*string {
	awsStringMap := make(map[string]*string)
	for k, v := range m {
		awsStringMap[k] = aws.String(v)
	}

	return awsStringMap
}

func convertStringPointerSlice(s []string) []*string {
	var stringPointerSlice []*string

	for _, str := range s {
		stringPointerSlice = append(stringPointerSlice, aws.String(str))
	}

	return stringPointerSlice
}

func getServiceFromARN(ARN string) string {
	return strings.Split(ARN, ":")[2]
}

func determineTagKeysToRemove(desiredTags, currentTags map[string]string) []string {
	var removeTags []string
	for k := range currentTags {
		if _, exists := desiredTags[k]; !exists {
			removeTags = append(removeTags, k)
		}
	}
	return removeTags
}

func NewResourceGroupsTagger(desiredAWSTags map[string]string) *ResourceGroupsTagger {
	if _, exists := desiredAWSTags[defaultAWSTagKey]; !exists {
		desiredAWSTags[defaultAWSTagKey] = defaultAWSTagValue
	}
	return &ResourceGroupsTagger{
		desiredAWSTags: desiredAWSTags,
	}
}
