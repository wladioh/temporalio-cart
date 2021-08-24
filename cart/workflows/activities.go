package workflows

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	propagation "services/cart/workflows/propagator"

	"github.com/go-resty/resty/v2"
	"github.com/google/uuid"
	"go.temporal.io/sdk/activity"
)

type Activities struct {
	HttpClient resty.Client
}

type RegisterPaymentApprovalRequest struct {
	TaskToken string
	CartId    string
}

func SampleActivity(ctx context.Context) (*propagation.Values, error) {
	if val := ctx.Value(propagation.PropagateKey); val != nil {
		vals := val.(propagation.Values)
		return &vals, nil
	}
	return nil, nil
}

func (client *Activities) RemovePaymentActivity(ctx context.Context, cartId uuid.UUID) error {
	logger := activity.GetLogger(ctx)
	resp, err := client.HttpClient.R().
		Delete("/payments/v1/" + cartId.String())

	logger.Info("registerCallback removed")
	if err != nil {
		logger.Info("RemovePaymentActivity failed to delete callback.", "Error", err)
		return err
	}
	status := resp.StatusCode()
	if resp.StatusCode() == 200 {
		// register callback succeed
		logger.Info("Successfully deleted callback.", "CartID", cartId)

		// ErrActivityResultPending is returned from activity's execution to indicate the activity is not completed when it returns.
		// activity will be completed asynchronously when Client.CompleteActivity() is called.
		return nil
	}

	logger.Warn("deleted callback failed.", "Cart Status", status)
	return fmt.Errorf("deleted callback failed status: %d", status)
}

func (client *Activities) WaitForPaymentActivity(ctx context.Context, cartId uuid.UUID) (string, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Starting WaitForPaymentActivity")
	if len(cartId) == 0 {
		return "", errors.New("Cart id is empty")
	}

	// save current activity info so it can be completed asynchronously when expense is approved/rejected
	activityInfo := activity.GetInfo(ctx)
	// t := string(activityInfo.TaskToken)
	// TaskToken := url.QueryEscape(t) //base64.URLEncoding.EncodeToString(activityInfo.TaskToken)

	t := url.QueryEscape(string(activityInfo.TaskToken))
	logger.Info("registerCallback calling")
	message := &RegisterPaymentApprovalRequest{TaskToken: t, CartId: cartId.String()}

	body, err := json.Marshal(message)
	resp, err := client.HttpClient.R().
		SetBody(body).
		Post("/payments/v1")

	logger.Info("registerCallback called")
	if err != nil {
		logger.Info("WaitForPaymentActivity failed to register callback.", "Error", err)
		return "", err
	}
	status := resp.StatusCode()
	if resp.StatusCode() == 200 {
		// register callback succeed
		logger.Info("Successfully registered callback.", "CartID", cartId)

		// ErrActivityResultPending is returned from activity's execution to indicate the activity is not completed when it returns.
		// activity will be completed asynchronously when Client.CompleteActivity() is called.
		return "", activity.ErrResultPending
	}

	logger.Warn("Register callback failed.", "Cart Status", status)
	return "", fmt.Errorf("register callback failed status: %d", status)
}
