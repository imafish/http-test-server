// Package common holds all internal types
// just a convenient way to organize code
package common

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// RequestHandler handles incoming requests
type RequestHandler struct {
	Rules []Rule
}

func (rh *RequestHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rule, err := FindMatchingRule(rh.Rules, r)

	if err != nil {
		ErrorResponse(http.StatusInternalServerError, fmt.Sprintf("error in finding matching rule for this request, err: %s", err.Error()), w)
		return
	}

	if rule == nil {
		ErrorResponse(http.StatusNotFound, "no matching rule found for this request", w)
	} else {
		WriteResponse(rule, w)
	}
}

// WriteResponse writes http responses base on rule's response field
func WriteResponse(rule *Rule, w http.ResponseWriter) {
	responseRule := rule.Response

	// headers
	for _, header := range responseRule.Headers {
		splits := strings.Split(header, ":")
		if len(splits) != 2 {
			ErrorResponse(http.StatusInternalServerError, fmt.Sprintf("header string should contain exact 1 colon, actual: %s", header), w)
			return
		}
		headerKey := strings.TrimSpace(splits[0])
		headerValue := strings.TrimSpace(splits[1])
		w.Header().Add(headerKey, headerValue)
	}

	// body
	filePath := responseRule.File
	objBody := responseRule.Body
	if filePath != "" {
		log.Printf("Creating file response using: %s", filePath)
		stat, err := os.Stat(filePath)
		if err != nil {
			ErrorResponse(http.StatusInternalServerError, fmt.Sprintf("Failed to find file, err: %s", err.Error()), w)
			return
		}

		inFile, err := os.Open(filePath)
		if err != nil {
			ErrorResponse(http.StatusInternalServerError, fmt.Sprintf("Failed to open file, err: %s", err), w)
			return
		}
		defer inFile.Close()

		// set headers and status code before write to body
		filename := filepath.Base(filePath)
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
		w.Header().Set("Content-Length", strconv.FormatInt(stat.Size(), 10))
		if responseRule.Status != 0 {
			w.WriteHeader(responseRule.Status)
		}

		buf := make([]byte, 1024)
		for {
			n, err := inFile.Read(buf)
			if err != nil && err != io.EOF {
				ErrorResponse(http.StatusInternalServerError, fmt.Sprintf("Failed to read file, err: %s", err.Error()), w)
				return
			}
			if n == 0 || err == io.EOF {
				break
			}
			w.Write(buf[:n])
		}

	} else if objBody != nil {
		log.Printf("Creating response body using object")
		jsonObj, err := convertToJSON(objBody)
		if err != nil {
			ErrorResponse(http.StatusInternalServerError, fmt.Sprintf("Failed to convert YAML object to JSON object, err: %s", err.Error()), w)
		}
		bytes, err := json.Marshal(jsonObj)
		if err != nil {
			ErrorResponse(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal obj into json, err: %s", err.Error()), w)
			return
		}

		// set status code before write to body
		if responseRule.Status != 0 {
			w.WriteHeader(responseRule.Status)
		}

		w.Write(bytes)
	}
}

func convertToJSON(objBody interface{}) (interface{}, error) {

	switch b := objBody.(type) {
	case map[interface{}]interface{}:
		result := make(map[string]interface{})
		for k, v := range b {
			keyString, ok := k.(string)
			if !ok {
				return nil, fmt.Errorf("key of map object must of string ")
			}

			vConverted, err := convertToJSON(v)
			if err != nil {
				return nil, err
			}
			result[keyString] = vConverted
		}
		return result, nil

	default:
		return b, nil
	}
}

// ErrorResponse reponses 500 status code, and body as errorMsg
func ErrorResponse(statusCode int, errorMsg string, w http.ResponseWriter) {
	log.Println(errorMsg)
	w.WriteHeader(statusCode)
	w.Write([]byte(errorMsg))
}
