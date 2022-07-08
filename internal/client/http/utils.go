package client

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/project-flotta/flotta-operator/models"
)

// extractData extracts data from the message content
func extractData[T any](res *http.Response, extractKey string) (T, error) {
	var result T

	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return result, fmt.Errorf("cannot read response body '%w'", err)
	}
	defer res.Body.Close()

	var message models.MessageResponse
	err = json.Unmarshal(data, &message)
	if err != nil {
		return result, fmt.Errorf("cannot read marshal body into message response '%w'", err)
	}

	content, ok := message.Content.(map[string]interface{})
	if !ok {
		return result, fmt.Errorf("payload content is not a map")
	}

	d, ok := content[extractKey]
	if !ok {
		return result, fmt.Errorf("cannot find configuration data in payload")
	}

	result, ok = d.(T)
	if !ok {
		return result, fmt.Errorf("cannot extract data from content. wrong type")
	}

	return result, nil
}
