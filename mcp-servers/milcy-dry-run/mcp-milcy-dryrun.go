package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http" // Corrected import path

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"gopkg.in/yaml.v2"
)

var protocol = "http"
var port = "8080"
var milcyDryrunServiceName = "ms-lifecycle-operator-dry-run-service"
var fqdnSuffix = ".svc.cluster.local"

// List of types to check
var resourceTypes = []string{
	"deployment",
	"service",
	"horizontalpodautoscaler",
	"amdocsapi",
	"serviceaccount",
	"resourcerequest",
	"amdocsservicemeshbinding",
	"amdocstrafficrouting",
}

// Structs for unmarshalling the response
type TypeInfo struct {
	Total int      `json:"total"`
	Names []string `json:"names"`
}

type ResponseBody struct {
	Types map[string]TypeInfo `json:"types"`
}

func main() {
	// Create a new MCP server
	log.Printf("Create a new MCP server")
	s := server.NewMCPServer(
		"yamldiff",
		"1.0.0",
		server.WithResourceCapabilities(true, true),
		server.WithLogging(),
	)

	// Add a Dryrun  tool
	dryrunTool := mcp.NewTool("dryrun",
		mcp.WithDescription("Perform dryrun using milcy dryrun"),
		mcp.WithString("resource",
			mcp.Required(),
			mcp.Description("input yaml resouce"),
		),
	)

	// Add the dry run handler
	s.AddTool(dryrunTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		resource := request.Params.Arguments["resource"].(string)

		log.Printf("input resource is: %s", resource)
		//check if the resource has kind and check the kind value
		if resource == "" {
			return nil, fmt.Errorf("resource is empty")
		}

		kind, err := extractKind(resource)
		if err != nil {
			return nil, fmt.Errorf("error extracting kind: %v", err)
		}
		name, err := extractName(resource)
		if err != nil {
			return nil, fmt.Errorf("error extracting name: %v", err)
		}
		namespace, err := extractNamespace(resource)
		if err != nil {
			return nil, fmt.Errorf("error extracting namespace: %v", err)
		}

		milcyDryrunServiceURL := fmt.Sprintf("%s://%s.%s%s:%s/dryrun", protocol, milcyDryrunServiceName, namespace, fqdnSuffix, port)
		log.Printf("---------------------------------------------------------")
		log.Printf("The resource name is: %s", name)
		log.Printf("The resource kind is: %s", kind)
		log.Printf("The resource namespace is: %s", namespace)
		log.Printf("milcy dry run service url is: %s", milcyDryrunServiceURL)
		log.Printf("The resource is: \n%s", resource)
		log.Printf("---------------------------------------------------------")

		if kind == "Microservice" {
			return reconcileMicroservice(resource, name, kind, milcyDryrunServiceURL)
		} else if kind == "ConfigBinding" {
			return reconcileConfigBinding(resource, name, kind, milcyDryrunServiceURL)
		} else if kind == "CustomResourceBinding" {
			reconcileCustomResourceBinding(resource, name, kind, milcyDryrunServiceURL)
		} else {
			return nil, fmt.Errorf("resource kind is not Microservice or ConfigBinding or CustomResourceBinding")
		}

		return nil, fmt.Errorf("resource kind is not Microservice or ConfigBinding or CustomResourceBinding")
	})

	//Start the server lon k8s
	sseServer := server.NewSSEServer(s, server.WithBaseURL("http://dryruntool-service.kagent.svc.cluster.local:8080"))
	log.Printf("SSE server listening on http://dryruntool-service.kagent.svc.cluster.local:8080")
	if err := sseServer.Start(":8080"); err != nil {
		log.Fatalf("Server error: %v", err)
	}

}

// reconcileMicroservice reconciles a microserivce resource.
func reconcileMicroservice(resource string, name string, kind string, milcyDryrunServiceURL string) (*mcp.CallToolResult, error) {
	// Implement the logic for reconciling a microservice
	log.Printf("reconcileMicroservice is called")

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)

	part, err := writer.CreateFormFile("file", "resource.yaml")
	if err != nil {
		return nil, fmt.Errorf("error creating form file: %v", err)
	}
	_, err = part.Write([]byte(resource))
	if err != nil {
		return nil, fmt.Errorf("error writing file content: %v", err)
	}

	err = writer.Close()
	if err != nil {
		return nil, fmt.Errorf("error closing writer: %v", err)
	}

	req, err := http.NewRequest("POST", milcyDryrunServiceURL+"/upload", &requestBody)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-Struct-Type", "Microservice")

	log.Printf("Upload microservice command is: URL=%s, Headers=%v, Body=%s", req.URL.String(), req.Header, requestBody.String())

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %v", err)
	}
	//print	the response status code
	log.Printf("Upload microservice -- Response status code: %d", resp.StatusCode)

	defer resp.Body.Close()

	//call the process endpoint
	req, err = http.NewRequest("GET", milcyDryrunServiceURL+"/process", nil)
	log.Printf("URL Called:%s", milcyDryrunServiceURL+"/process")
	if err != nil {
		return nil, fmt.Errorf("error creating request /process: %v", err)
	}
	resp, err = client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request /process: %v", err)
	}
	log.Printf("Process API -- Response status code: %d", resp.StatusCode)
	defer resp.Body.Close()

	//call the status endpoint
	req, err = http.NewRequest("GET", milcyDryrunServiceURL+"/status/microservice/"+name, nil)
	log.Printf("URL Called: %s", milcyDryrunServiceURL+"/status/microservice/%s", name)
	if err != nil {
		return nil, fmt.Errorf("error creating request /status/microservice/%s: %v", name, err)
	}
	resp, err = client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request /status/microservice/%s: %v", name, err)
	}
	log.Printf("Status API -- Response status code: %d", resp.StatusCode)

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading response body: %v", err)
		return nil, fmt.Errorf("error reading response body: %v", err)
	}
	log.Printf("Response body for calling status endpoint is: %s", string(bodyBytes))
	//printResourceNames(bodyBytes)
	resourceMap, err := getResourceNamesMap(bodyBytes)
	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		log.Printf("Resource map: %+v", resourceMap)
	}

	defer resp.Body.Close()
	var result = ""
	//iterate over the resource map and print the resource names
	for resourceType, names := range resourceMap {
		log.Printf("Resource type: %s", resourceType)
		for _, name := range names {
			log.Printf("Start Handling Resource name: %s", name)
			req, err = http.NewRequest("GET", milcyDryrunServiceURL+"/get/"+resourceType+"/"+name, nil)
			log.Printf("URL Called: %s", milcyDryrunServiceURL+"/get/"+resourceType+"/"+name)
			if err != nil {
				return nil, fmt.Errorf("error creating request /get/%s/%s: %v", resourceType, name, err)
			}

			resp, err = client.Do(req)
			if err != nil {
				return nil, fmt.Errorf("error sending request /get/%s/%s: %v", resourceType, name, err)
			}
			log.Printf("Get resource from cache -- Response status code: %d", resp.StatusCode)

			cacheResource, err := io.ReadAll(resp.Body)
			if err != nil {
				log.Printf("Error reading response body from cache: %v", err)
				return nil, fmt.Errorf("error reading response body: %v", err)
			}
			defer resp.Body.Close()

			req, err = http.NewRequest("GET", milcyDryrunServiceURL+"/get/"+resourceType+"/"+name+"/direct", nil)
			log.Printf("URL Called: %s", milcyDryrunServiceURL+"/get/"+resourceType+"/"+name+"/direct")
			if err != nil {
				return nil, fmt.Errorf("error creating request /get/%s/%s/direct: %v", resourceType, name, err)
			}
			resp, err = client.Do(req)
			if err != nil {
				return nil, fmt.Errorf("error sending request /get/%s/%s/direct: %v", resourceType, name, err)
			}

			log.Printf("Get MS from k8s -- Response status code: %d", resp.StatusCode)
			k8sResource, err := io.ReadAll(resp.Body)
			if err != nil {
				log.Printf("Error reading response body from k8s: %v", err)
				return nil, fmt.Errorf("error reading response body: %v", err)
			}

			defer resp.Body.Close()

			yaml1, err := extractSpec(string(cacheResource))
			if err != nil {
				log.Printf("Error extracting spec from cacheResource: %v", err)
				return nil, fmt.Errorf("error extracting spec from cacheResource: %v", err)
			}
			yaml2, err := extractSpec(string(k8sResource))
			if err != nil {
				log.Printf("Error extracting spec from k8sResource: %v", err)
				return nil, fmt.Errorf("error extracting spec from k8sResource: %v", err)
			}

			// Combine yaml1 and yaml2 with a separator
			result = result + fmt.Sprintf("YAML1:\n%s\n\nYAML2:\n%s", yaml1, yaml2)
			result += "\nEnd Of Yaml\n"
			//log.Printf("result is: \n%s", result)
		}
	}
	//log.Printf("Final result is: \n%s", result)
	return mcp.NewToolResultText(result), nil

}

func reconcileConfigBinding(resource string, name string, kind string, milcyDryrunServiceURL string) (*mcp.CallToolResult, error) {
	// Implement the logic for reconciling a config binding
	log.Printf("reconcileConfigBinding is called")

	extractMsSelector(resource)
	// Check if the msSelector is empty
	// if msSelector is empty, return error
	// if msSelector is not empty, continue
	msSelector, err := extractMsSelector(resource)
	if err != nil {
		return nil, fmt.Errorf("error extracting msSelector: %v", err)
	}
	msName := ""
	if len(msSelector) == 0 {
		log.Printf("msSelector is empty no impact of deplying config binding")
		return nil, nil
	} else {
		//check if we have key and value in msSelector and the key name is "name"
		for key, value := range msSelector {
			if key == "name" {
				log.Printf("msSelector is not empty and the key name is: %s", value)
				msName = value.(string)
			} else {
				return nil, fmt.Errorf("msSelector is not empty and the key name is not name")
			}
		}
	}

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)

	part, err := writer.CreateFormFile("file", "resource.yaml")
	if err != nil {
		return nil, fmt.Errorf("error creating form file: %v", err)
	}
	_, err = part.Write([]byte(resource))
	if err != nil {
		return nil, fmt.Errorf("error writing file content: %v", err)
	}

	err = writer.Close()
	if err != nil {
		return nil, fmt.Errorf("error closing writer: %v", err)
	}

	req, err := http.NewRequest("POST", milcyDryrunServiceURL+"/upload", &requestBody)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-Struct-Type", "ConfigBinding")

	//print the request command and headers
	log.Printf("Upload configbinding command is: URL=%s, Headers=%v, Body=%s", req.URL.String(), req.Header, requestBody.String())

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	//call the process endpoint
	req, err = http.NewRequest("GET", milcyDryrunServiceURL+"/process", nil)
	log.Printf("URL Called: ", milcyDryrunServiceURL+"/process")
	if err != nil {
		return nil, fmt.Errorf("error creating request /process: %v", err)
	}
	resp, err = client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request /process: %v", err)
	}
	defer resp.Body.Close()

	// Get the cache resource
	req, err = http.NewRequest("GET", milcyDryrunServiceURL+"/get/deployment/"+msName, nil)
	log.Printf("URL Called: %s", milcyDryrunServiceURL+"/get/deployment/"+msName)
	if err != nil {
		return nil, fmt.Errorf("error creating request /get/deployment/%s: %v", msName, err)
	}
	resp, err = client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request /get/deployment/%s: %v", msName, err)
	}
	defer resp.Body.Close()

	cacheResource, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %v", err)
	}

	// Get the k8s resource
	req, err = http.NewRequest("GET", milcyDryrunServiceURL+"/get/deployment/"+msName+"/direct", nil)

	log.Printf("URL Called: %s", milcyDryrunServiceURL+"/get/deployment/"+msName+"/direct")
	if err != nil {
		return nil, fmt.Errorf("error creating request /get/deployment/%s/direct: %v", msName, err)
	}
	resp, err = client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request /get/deployment/%s/direct: %v", msName, err)
	}

	defer resp.Body.Close()
	k8sResource, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %v", err)
	}

	yaml1 := string(cacheResource)
	yaml2 := string(k8sResource)
	// Combine yaml1 and yaml2 with a separator
	result := fmt.Sprintf("YAML1:\n%s\n\nYAML2:\n%s", yaml1, yaml2)
	log.Printf("cacheResource is: \n%s", yaml1)
	log.Printf("k8sResource is: \n%s", yaml2)
	return mcp.NewToolResultText(result), nil
}

// reconcileCustomResourceBinding reconciles a custom resource binding resource.
func reconcileCustomResourceBinding(resource string, name string, kind string, milcyDryrunServiceURL string) (*mcp.CallToolResult, error) {
	// Implement the logic for reconciling a custom resource binding
	log.Printf("reconcileCustomResourceBinding is called")

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)

	part, err := writer.CreateFormFile("file", "resource.yaml")
	if err != nil {
		return nil, fmt.Errorf("error creating form file: %v", err)
	}
	_, err = part.Write([]byte(resource))
	if err != nil {
		return nil, fmt.Errorf("error writing file content: %v", err)
	}

	err = writer.Close()
	if err != nil {
		return nil, fmt.Errorf("error closing writer: %v", err)
	}

	req, err := http.NewRequest("POST", milcyDryrunServiceURL+"/upload", &requestBody)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-Struct-Type", "CustomResourceBinding")

	//print the request command and headers
	log.Printf("Upload CustomResourceBinding command is: URL=%s, Headers=%v, Body=%s", req.URL.String(), req.Header, requestBody.String())

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	req, err = http.NewRequest("GET", milcyDryrunServiceURL+"/get/customresourcebinding/"+name, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request /get/customresourcebinding/%s: %v", name, err)
	}
	resp, err = client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request /get/customresourcebinding/%s: %v", name, err)
	}
	defer resp.Body.Close()

	cacheResource, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %v", err)
	}

	req, err = http.NewRequest("GET", milcyDryrunServiceURL+"/get/customresourcebinding/"+name+"/direct", nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request /get/customresourcebinding/%s/direct: %v", name, err)
	}
	resp, err = client.Do(req)
	if err != nil {
		return nil,
			fmt.Errorf("error sending request /get/customresourcebinding/%s/direct: %v", name, err)
	}
	defer resp.Body.Close()
	k8sResource, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %v", err)
	}
	yaml1 := string(cacheResource)
	yaml2 := string(k8sResource)
	// Combine yaml1 and yaml2 with a separator
	result := fmt.Sprintf("YAML1:\n%s\n\nYAML2:\n%s", yaml1, yaml2)
	log.Printf("cacheResource is: %s", yaml1)
	log.Printf("k8sResource is: %s", yaml2)
	return mcp.NewToolResultText(result), nil
}

// ExtractKind extracts the "kind" field from a YAML resource.
func extractKind(resource string) (string, error) {
	// Define a map to parse the YAML into
	var parsedResource map[string]interface{}

	// Unmarshal the YAML into the map
	err := yaml.Unmarshal([]byte(resource), &parsedResource)
	if err != nil {
		return "", fmt.Errorf("failed to parse YAML: %v", err)
	}

	// Check if the "kind" field exists
	kind, ok := parsedResource["kind"].(string)
	if !ok {
		return "", fmt.Errorf("kind field not found or is not a string")
	}

	return kind, nil
}

// Exract spec section from the resource and return it as string
func extractSpec(resource string) (string, error) {
	// Define a map to parse the YAML into
	var parsedResource map[string]interface{}

	// Unmarshal the YAML into the map
	err := yaml.Unmarshal([]byte(resource), &parsedResource)
	if err != nil {
		return "", fmt.Errorf("failed to parse YAML: %v", err)
	}

	// Check if the "spec" section exists
	spec, ok := parsedResource["spec"].(map[interface{}]interface{})
	if !ok {
		return "", fmt.Errorf("spec section not found or is not a map")
	}

	// Marshal the spec back to YAML
	specYAML, err := yaml.Marshal(spec)
	if err != nil {
		return "", fmt.Errorf("failed to marshal spec to YAML: %v", err)
	}

	return string(specYAML), nil
}

// extractName extracts the "name" field from a YAML resource under the "metadata" section.
func extractName(resource string) (string, error) {
	// Define a map to parse the YAML into
	var parsedResource map[string]interface{}

	// Unmarshal the YAML into the map
	err := yaml.Unmarshal([]byte(resource), &parsedResource)
	if err != nil {
		return "", fmt.Errorf("failed to parse YAML: %v", err)
	}

	// Check if the "metadata" section exists
	metadata, ok := parsedResource["metadata"].(map[interface{}]interface{})
	if !ok {
		return "", fmt.Errorf("metadata section not found or is not a map")
	}

	// Check if the "name" field exists in the "metadata" section
	name, ok := metadata["name"].(string)
	if !ok {
		return "", fmt.Errorf("name field not found or is not a string")
	}

	return name, nil
}

// extractNamespace extracts the "namespace" field from a YAML resource under the "metadata" section.
func extractNamespace(resource string) (string, error) {
	// Define a map to parse the YAML into
	var parsedResource map[string]interface{}

	// Unmarshal the YAML into the map
	err := yaml.Unmarshal([]byte(resource), &parsedResource)
	if err != nil {
		return "", fmt.Errorf("failed to parse YAML: %v", err)
	}

	// Check if the "metadata" section exists
	metadata, ok := parsedResource["metadata"].(map[interface{}]interface{})
	if !ok {
		return "", fmt.Errorf("metadata section not found or is not a map")
	}

	// Check if the "namespace" field exists in the "metadata" section
	namespace, ok := metadata["namespace"].(string)
	if !ok {
		return "default", nil // Default to "default" namespace if not found
	}

	return namespace, nil
}

// extractMsSelector extracts the "msSelector" field from a YAML resource under the "spec" section.
func extractMsSelector(resource string) (map[string]interface{}, error) {
	// Define a map to parse the YAML into
	var parsedResource map[string]interface{}

	// Unmarshal the YAML into the map
	err := yaml.Unmarshal([]byte(resource), &parsedResource)
	if err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %v", err)
	}

	// Check if the "spec" section exists
	spec, ok := parsedResource["spec"].(map[interface{}]interface{})
	if !ok {
		return nil, fmt.Errorf("spec section not found or is not a map")
	}

	// Check if the "msSelector" field exists in the "spec" section
	msSelector, ok := spec["msSelector"].(map[interface{}]interface{})
	if !ok {
		return nil, fmt.Errorf("msSelector field not found or is not a map")
	}

	// Convert msSelector to a map[string]interface{} for easier handling
	result := make(map[string]interface{})
	for key, value := range msSelector {
		strKey, ok := key.(string)
		if !ok {
			return nil, fmt.Errorf("msSelector key is not a string")
		}
		result[strKey] = value
	}

	return result, nil
}

// Method to print resource names for specified types
func printResourceNames(body []byte) error {
	log.Printf("printResourceNames is called")
	var resp ResponseBody
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("failed to unmarshal response body: %v", err)
	}
	for _, t := range resourceTypes {
		if info, ok := resp.Types[t]; ok {
			for _, name := range info.Names {
				log.Printf("Type: %s, Name: %s\n", t, name)
			}
		}
	}
	return nil
}

// Method to return resource names for specified types as a map
func getResourceNamesMap(body []byte) (map[string][]string, error) {
	log.Printf("getResourceNamesMap is called")
	var resp ResponseBody
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response body: %v", err)
	}
	result := make(map[string][]string)
	for _, t := range resourceTypes {
		if info, ok := resp.Types[t]; ok {
			result[t] = info.Names
		}
	}
	return result, nil
}
