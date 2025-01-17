package waiter

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudcontrolapi"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/terraform-providers/terraform-provider-aws/aws/internal/tfresource"
)

func ProgressEventOperationStatusSuccess(ctx context.Context, conn *cloudcontrolapi.CloudControlApi, requestToken string, timeout time.Duration) (*cloudcontrolapi.ProgressEvent, error) {
	stateConf := &resource.StateChangeConf{
		Pending: []string{cloudcontrolapi.OperationStatusInProgress, cloudcontrolapi.OperationStatusPending},
		Target:  []string{cloudcontrolapi.OperationStatusSuccess},
		Refresh: ProgressEventOperationStatus(ctx, conn, requestToken),
		Timeout: timeout,
	}

	outputRaw, err := stateConf.WaitForStateContext(ctx)

	if output, ok := outputRaw.(*cloudcontrolapi.ProgressEvent); ok {
		if operationStatus := aws.StringValue(output.OperationStatus); operationStatus == cloudcontrolapi.OperationStatusFailed {
			tfresource.SetLastError(err, fmt.Errorf("%s: %s", aws.StringValue(output.ErrorCode), aws.StringValue(output.StatusMessage)))
		}

		return output, err
	}

	return nil, err
}
