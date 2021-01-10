package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/imafish/http-test-server/internal/rules"
)

// RequestHandler handles incoming requests
type RequestHandler struct {
	Rules *[]*rules.CompiledRule
	Mtx   *sync.Mutex
}

func (rh *RequestHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	log.Println("\n------- ------- -------")
	log.Printf("Incoming request: %s", r.RequestURI)
	bodyBytes, _ := ioutil.ReadAll(r.Body)
	r.Body.Close()
	r.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))
	log.Printf("Incoming request body: %s", string(bodyBytes))

	rh.Mtx.Lock()
	defer rh.Mtx.Unlock()

	rule, variables, err := rules.FindMatchingRule(rh.Rules, r)
	if err != nil {
		errorResponse(http.StatusInternalServerError, fmt.Sprintf("error in finding matching rule for this request, err: %s", err.Error()), w)
		return
	}
	if rule == nil {
		errorResponse(http.StatusNotFound, "no matching rule found for this request", w)
	} else {
		log.Printf("Found rule '%s'", rule.Name)
		writeResponse(rule, variables, w)
	}
}

func writeResponse(rule *rules.CompiledRule, variables map[string]*rules.Variable, w http.ResponseWriter) {
	responseRule := rule.Response

	// headers
	for _, header := range responseRule.Headers {
		splits := strings.Split(header, ":")
		if len(splits) != 2 {
			errorResponse(http.StatusInternalServerError, fmt.Sprintf("header string should contain exact 1 colon, actual: %s", header), w)
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
			errorResponse(http.StatusInternalServerError, fmt.Sprintf("Failed to find file, err: %s", err.Error()), w)
			return
		}

		inFile, err := os.Open(filePath)
		if err != nil {
			errorResponse(http.StatusInternalServerError, fmt.Sprintf("Failed to open file, err: %s", err), w)
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
				errorResponse(http.StatusInternalServerError, fmt.Sprintf("Failed to read file, err: %s", err.Error()), w)
				return
			}
			if n == 0 || err == io.EOF {
				break
			}
			w.Write(buf[:n])
		}

	} else if objBody != nil {
		log.Printf("Creating response body using object")
		jsonObj, err := convertToJSON(objBody, variables)
		if err != nil {
			errorResponse(http.StatusInternalServerError, fmt.Sprintf("Failed to convert YAML object to JSON object, err: %s", err.Error()), w)
		}
		bytes, err := json.Marshal(jsonObj)
		if err != nil {
			errorResponse(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal obj into json, err: %s", err.Error()), w)
			return
		}

		// set status code before write to body
		if responseRule.Status != 0 {
			w.WriteHeader(responseRule.Status)
		}

		w.Write(bytes)
	}
}

func convertToJSON(objBody interface{}, variables map[string]*rules.Variable) (interface{}, error) {

	switch b := objBody.(type) {
	case map[interface{}]interface{}:
		result := make(map[string]interface{})
		for k, v := range b {
			keyString, ok := k.(string)
			if !ok {
				return nil, fmt.Errorf("key of map object must of string ")
			}

			vConverted, err := convertToJSON(v, variables)
			if err != nil {
				return nil, err
			}
			result[keyString] = vConverted
		}
		return result, nil

	case []interface{}:
		result := make([]interface{}, 0, len(b))
		for _, v := range b {
			vConverted, err := convertToJSON(v, variables)
			if err != nil {
				return nil, err
			}
			result = append(result, vConverted)
		}
		return result, nil

	case string:
		regex := regexp.MustCompile(`^{{(\w+)}}$`)
		matches := regex.FindStringSubmatch(b)
		if matches != nil {
			// single match
			v := variables[matches[1]]
			if v == nil {
				return nil, nil
			}
			return v.GetValue()
		}

		for k, v := range variables {
			regex, err := regexp.Compile("{{" + k + "}}")
			if err != nil {
				return nil, err
			}

			value, err := v.GetValue()
			if err != nil {
				return nil, err
			}
			b = regex.ReplaceAllString(b, fmt.Sprint(value))
		}

		return b, nil

	default:
		return b, nil
	}
}

// errorResponse reponses 500 status code, and body as errorMsg
func errorResponse(statusCode int, errorMsg string, w http.ResponseWriter) {
	log.Println(errorMsg)
	w.WriteHeader(statusCode)
	w.Write([]byte(errorMsg))
}
