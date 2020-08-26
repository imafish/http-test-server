// Package common holds all internal types
// just a convenient way to organize code
package common

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
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

	// status code
	if responseRule.Status == 0 {
		w.WriteHeader(200)
	} else {
		w.WriteHeader(responseRule.Status)
	}

	// header
	for _, header := range responseRule.Headers {
		splits := strings.Split(header, ":")
		if len(splits) != 2 {
			log.Printf("header string should contain exact 1 colon, actual: %s", header)
			continue
		}
		w.Header()[strings.TrimSpace(splits[0])] = []string{strings.TrimSpace(splits[1])}
	}

	// body
	filePath := responseRule.File
	objBody := responseRule.Body
	if filePath != "" {
		log.Printf("Creating file response using: %s", filePath)
		if _, err := os.Stat(filePath); err != nil {
			ErrorResponse(http.StatusInternalServerError, fmt.Sprintf("Failed to find file, err: %s", err.Error()), w)
			return
		}

		inFile, err := os.Open(filePath)
		if err != nil {
			ErrorResponse(http.StatusInternalServerError, fmt.Sprintf("Failed to open file, err: %s", err), w)
			return
		}
		defer inFile.Close()

		buf := make([]byte, 1024)
		contentLength := 0
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
			contentLength += n
		}

		filename := filepath.Base(filePath)
		w.Header()["Content-Disposition"] = []string{fmt.Sprintf("attachment; filename=\"%s\"", filename)}
		w.Header()["Content-Length"] = []string{strconv.Itoa(contentLength)}

	} else if objBody != nil {
		bytes, err := json.Marshal(objBody)
		if err != nil {
			ErrorResponse(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal obj into json, err: %s", err.Error()), w)
			return
		}

		w.Write(bytes)
	}
}

// ErrorResponse reponses 500 status code, and body as errorMsg
func ErrorResponse(statusCode int, errorMsg string, w http.ResponseWriter) {
	log.Println(errorMsg)
	w.WriteHeader(statusCode)
	w.Write([]byte(errorMsg))
}
