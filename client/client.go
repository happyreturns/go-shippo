package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/d5/go-shippo/models"
)

const shippoAPIBaseURL = "https://api.goshippo.com/v1"

type Client struct {
	privateToken string
}

type listOutputCallback func(v json.RawMessage) error

// NewClient creates a new Shippo API client instance.
func NewClient(privateToken string) *Client {
	return &Client{
		privateToken: privateToken,
	}
}

func (c *Client) do(method, path string, input, output interface{}) error {
	url := shippoAPIBaseURL + path

	req, err := c.createRequest(method, url, input)
	if err != nil {
		return fmt.Errorf("Error creating request object: %s", err.Error())
	}

	status, err := c.executeRequest(req, output)
	if err != nil {
		return fmt.Errorf("Error executing request: [%d] %s", status, err.Error())
	}
	if status < 200 || status >= 300 {
		return fmt.Errorf("Error status returned: %d", status)
	}

	return nil
}

func (c *Client) doList(method, path string, input interface{}, outputCallback listOutputCallback) error {
	nextURL := shippoAPIBaseURL + path + "?results=25"

	for {
		req, err := c.createRequest(method, nextURL, input)
		if err != nil {
			return fmt.Errorf("Error creating request object: %s", err.Error())
		}

		listOutput := &models.ListAPIOutput{}
		status, err := c.executeRequest(req, listOutput)
		if err != nil {
			return fmt.Errorf("Error executing request: [%d] %s", status, err.Error())
		}
		if status < 200 || status >= 300 {
			return fmt.Errorf("Error status returned: %d", status)
		}

		for _, v := range listOutput.Results {
			if err := outputCallback(v); err != nil {
				return fmt.Errorf("Error unmarshalling output item: %s", err.Error())
			}
		}

		if listOutput.NextPageURL == nil {
			break
		}

		nextURL = *listOutput.NextPageURL
	}

	return nil
}

func (c *Client) createRequest(method, url string, bodyObject interface{}) (*http.Request, error) {
	var reqBody io.Reader
	if bodyObject != nil {
		data, err := json.Marshal(bodyObject)
		if err != nil {
			return nil, fmt.Errorf("Error marshaling body object: %s", err.Error())
		}

		reqBody = bytes.NewBuffer(data)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("Error creating HTTP request: %s", err.Error())
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "ShippoToken "+c.privateToken)

	// no keep-alive
	req.Header.Set("Connection", "close")
	req.Close = true

	return req, nil
}

func (c *Client) executeRequest(req *http.Request, output interface{}) (int, error) {
	httpClient := http.Client{}

	res, err := httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("Error making HTTP request: %s", err.Error())
	}
	defer res.Body.Close()

	resData, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return res.StatusCode, fmt.Errorf("Error reading response body data: %s", err.Error())
	}

	if output != nil && len(resData) > 0 {
		if err := json.Unmarshal(resData, output); err != nil {
			return res.StatusCode, fmt.Errorf("Error unmarshaling response data: %s", err.Error())
		}
	}

	return res.StatusCode, nil
}
