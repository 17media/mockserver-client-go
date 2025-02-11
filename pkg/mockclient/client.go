package mockclient

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"testing"

	"net/http"
	"net/url"
	"strings"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Client wraps the testing context used for all interactions with MockServer
type Client struct {
	// testing type T
	T *testing.T
	// mock server base address
	BaseURL string
}

// AddExpectation adds an expectation based on a request matcher to MockServer
func (c *Client) AddExpectation(exp *Expectation) {
	msg, err := json.Marshal(exp)
	if err != nil {
		require.NoError(c.T, err,
			"Failed to serialize mock server expectation.")
	}

	c.callMock("expectation", string(msg))
}

// AddVerification adds a verification of requests to MockServer
func (c *Client) AddVerification(exp *Expectation) {
	msg, err := json.Marshal(exp)
	if err != nil {
		require.NoError(c.T, err,
			"Failed to serialize mock server verification.")
	}

	c.callMock("verify", string(msg))
}

// AddVerificationSequence adds a verification of a specific sequence of requests to MockServer
func (c *Client) AddVerificationSequence(exps ...*Expectation) {
	vs := &VerificationSequence{}
	for _, exp := range exps {
		// Only request part of the expectation will be used for verification sequences
		vs.Requests = append(vs.Requests, exp.Request)
	}
	msg, err := json.Marshal(vs)
	if err != nil {
		require.NoError(c.T, err,
			"Failed to serialize mock server verification sequence.")
	}

	c.callMock("verifySequence", string(msg))
}

// Clear everything that matches a given path in MockServer
func (c *Client) Clear(path string) {
	mockReqBody := fmt.Sprintf(`
			{
				"path": "%s"
			}
			`, path)
	c.callMock("clear", mockReqBody)
}

// Reset the entire MockServer, clearing all state
func (c *Client) Reset() {
	c.callMock("reset", "")
}

func (c *Client) callMock(mockAPI, mockReqBody string) {
	mockURL := fmt.Sprintf("%s/%s", c.BaseURL, mockAPI)
	// check url is valid
	if _, err := url.ParseRequestURI(mockURL); err != nil {
		require.NoError(c.T, err,
			fmt.Sprintf("'%s' is not a valid mock server URL", mockURL))
	}

	hc := &http.Client{
		// No timeout
	}
	reader := strings.NewReader(mockReqBody)

	mockReq, err := http.NewRequest("PUT", mockURL, reader)
	if err != nil {
		require.NoError(c.T, err, "Failed to create request to mock server.")
	}
	mockRes, err := hc.Do(mockReq)
	if err != nil {
		require.NoError(c.T, err, "Failed to send request to MockServer.")
	}

	// all went well so return (clears & /reset return 200 whilst /verify & /expectation return 201)
	if mockRes.StatusCode >= 200 && mockRes.StatusCode <= 299 {
		return
	}

	// something went wrong so return the error message (Note: MockServer /verify returns 406 on failure)
	b, err := ioutil.ReadAll(mockRes.Body)
	if err != nil {
		assert.Fail(c.T, fmt.Sprintf("MockServer call failed with status: %s", mockRes.Status))
	}
	assert.Fail(c.T, fmt.Sprintf("MockServer call to /%s failed with status: %s. Error message from MockServer is: %s", mockAPI, mockRes.Status, string(b)))
}
