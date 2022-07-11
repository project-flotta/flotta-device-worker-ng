package client

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/project-flotta/flotta-operator/models"
)

// extractData extracts data from the message content by looking into message.Content map and
// applying a custom transformation function on the extracted data.
func extractData[T, S any](response *http.Response, extractKey string, tranformFunc func(t T) (S, error)) (S, error) {
	var (
		result S
		res    T
	)

	if tranformFunc == nil {
		return result, fmt.Errorf("tranformFunc is missing")
	}

	data, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return result, fmt.Errorf("cannot read response body '%w'", err)
	}
	defer response.Body.Close()

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

	res, ok = d.(T)
	if !ok {
		return result, fmt.Errorf("cannot extract data from content. wrong type")
	}

	// apply custom transformation function on the result
	result, err = tranformFunc(res)
	if err != nil {
		return result, fmt.Errorf("error applying transformation function %w", err)
	}

	return result, nil
}

func transformToConfiguration(data map[string]interface{}) (models.DeviceConfiguration, error) {
	var result models.DeviceConfiguration

	j, err := json.Marshal(data)
	if err != nil {
		return result, fmt.Errorf("cannot marshal data: '%w'", err)
	}

	err = json.Unmarshal(j, &result)
	if err != nil {
		return result, fmt.Errorf("cannot unmarshal configuration: '%w'", err)
	}

	return result, nil
}
