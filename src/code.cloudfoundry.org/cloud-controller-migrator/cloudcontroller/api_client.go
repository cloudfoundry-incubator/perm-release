package cloudcontroller

import (
	"bytes"
	"context"
	"encoding/json"
	"io"

	"net/http"

	"fmt"

	"code.cloudfoundry.org/cloud-controller-migrator/messages"
	"code.cloudfoundry.org/lager"
)

type APIClient struct {
	Host       string
	HTTPClient *http.Client
}

func NewAPIClient(host string, client *http.Client) *APIClient {
	return &APIClient{
		Host:       host,
		HTTPClient: client,
	}
}

func (c *APIClient) MakePaginatedGetRequest(ctx context.Context, logger lager.Logger, route string, bodyCallback func(context.Context, lager.Logger, io.Reader) error) error {
	rg := NewRequestGenerator(c.Host)

	var (
		res *http.Response
		err error

		paginatedResponse PaginatedResponse

		routeLogger lager.Logger
	)

	for {
		routeLogger = logger.WithData(lager.Data{
			"route": route,
		})

		res, err = makeAPIRequest(ctx, routeLogger.Session("make-api-request"), c.HTTPClient, rg, route)
		if err != nil {
			return err
		}

		var body []byte
		buf := bytes.NewBuffer(body)
		r := io.TeeReader(res.Body, buf)

		defer res.Body.Close()

		err = json.NewDecoder(r).Decode(&paginatedResponse)
		if err != nil {
			return err
		}

		err = bodyCallback(ctx, routeLogger, buf)
		if err != nil {
			return err
		}

		if paginatedResponse.NextURL == nil {
			break
		} else {
			route = *paginatedResponse.NextURL
		}
	}

	return nil
}

func makeAPIRequest(ctx context.Context, logger lager.Logger, client *http.Client, rg *RequestGenerator, route string) (*http.Response, error) {
	req, err := rg.NewGetRequest(logger.Session("new-get-request"), route)
	if err != nil {
		return nil, err
	}

	req = req.WithContext(ctx)

	logger.Debug("making-request")
	res, err := client.Do(req)
	if err != nil {
		logger.Error(messages.FailedToPerformRequest, err)
		return nil, err
	}

	if res.StatusCode >= 400 {
		err = fmt.Errorf("HTTP bad response: %d", res.StatusCode)
		logger.Error("failed-to-ping-cloudcontroller", err)
		return nil, err
	}

	return res, nil
}
