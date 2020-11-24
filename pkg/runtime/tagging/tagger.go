package tagging

import (
	ackv1alpha1 "github.com/aws/aws-controllers-k8s/apis/core/v1alpha1"
	"github.com/aws/aws-sdk-go/aws/session"
)

type Tagger interface {
	SyncTags(ARN *ackv1alpha1.AWSResourceName, session *session.Session) error
}
