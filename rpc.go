package main

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

var (
	// Create a global HTTP client
	rpcClient     *http.Client
	rpcClientOnce sync.Once

	// Server information
	rpcIpAddr        string
	rpcPort          int
	rpcBasicAuthUser string
	rpcBasicAuthPass string
	rpcSecure        bool = false
)

type MoneroRPCRequest struct {
	Jsonrpc string                 `json:"jsonrpc"`
	Method  string                 `json:"method"`
	Params  map[string]interface{} `json:"params"`
	ID      int                    `json:"id"`
}

type MoneroRPCResponse struct {
	Result interface{} `json:"result"`
}

/*
* Function to dial the Monero RPC server
*
* @param cert_path: The path to the certificate file (optional)
* @param server: The Monero RPC server IP address
* @param port: The Monero RPC server port
* @param username: The username for basic auth, if any (optional)
* @param password: The password for basic auth, if any (optional)
*
* @return http.Client: The HTTP client connected to the Monero RPC server
 */
func dial_monero_rpc(cert_path string, server string, port int, username string, password string) http.Client {

	// Use sync.Once to ensure that the client is created only once
	rpcClientOnce.Do(func() {
		var tlsConfig *tls.Config

		// Load the self-signed certificate
		if cert_path != "" {
			log.Println("Secure connection enabled, loading CA certificate from path: ", cert_path)

			caCert, err := os.ReadFile(cert_path)
			if err != nil {
				log.Fatalf("Failed to read CA certificate: %v", err)
			}

			// Create a certificate pool from the certificate
			// and append the certificate to the pool of trusted certificates
			caCertPool := x509.NewCertPool()
			caCertPool.AppendCertsFromPEM(caCert)

			log.Println("Added custom CA certificate to the certificate pool")
			tlsConfig = &tls.Config{
				RootCAs:            caCertPool,
				InsecureSkipVerify: true,
			}

			rpcSecure = true
		}

		// Create an HTTP client with the TLS configuration,
		// tlsConfig will be nil if secure is false, which will
		// load the default system certificate pool and
		// connect to the Monero RPC server without custom certificates,
		// which is fine if server is insecure or the certificate is signed by
		// a trusted CA
		rpcClient = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: tlsConfig,
			},
			Timeout: 10 * time.Second,
		}

		log.Println("Initialized a new HTTP client")

		// Check if basic auth credentials are provided
		if username != "" && password != "" {
			rpcBasicAuthUser = username
			rpcBasicAuthPass = password
			log.Println("Basic auth credentials received, setting up basic auth")
		}

		// Set the server information
		rpcIpAddr = server
		rpcPort = port

		// Use the client to make a request to the Monero RPC server to test the connection
		newRequest := MoneroRPCRequest{
			Jsonrpc: "2.0",
			Method:  "get_info",
			Params:  map[string]interface{}{},
			ID:      1,
		}

		resp, err := make_rpc_request(newRequest)
		if err != nil {
			log.Fatalf("Failed to connect to Monero RPC server: %v", err)
		}
		defer resp.Body.Close()

		var result MoneroRPCResponse
		err = json.NewDecoder(resp.Body).Decode(&result)
		if err != nil {
			log.Fatalf("Failed to decode response body: %v", err)
		}

		log.Println(result.Result)

		log.Println("Connected to Monero RPC server successfully", resp.Status)
	})

	return *rpcClient
}

/*
* Function to make an RPC request to the Monero RPC server
*
* @param request: The MoneroRPCRequest object containing the request parameters
 */
func make_rpc_request(request MoneroRPCRequest) (*http.Response, error) {
	if rpcClient == nil {
		log.Fatal("HTTP client not initialized, please dial the Monero RPC server first")
	}

	rpcPortStr := strconv.Itoa(rpcPort)

	// Encode the parameters
	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	protocol := "http"
	if rpcSecure {
		protocol = "https"
	}

	address := protocol + "://" + rpcIpAddr + ":" + rpcPortStr + "/json_rpc"
	log.Println("Making RPC request to: ", address)

	// Create a new request
	req, err := http.NewRequest("GET", address, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if rpcBasicAuthUser != "" && rpcBasicAuthPass != "" {
		log.Println("Setting basic auth for the request, user:", rpcBasicAuthUser)

		// Encode the basic auth credentials to base64
		encodedAuth := base64.StdEncoding.EncodeToString([]byte(rpcBasicAuthUser + ":" + rpcBasicAuthPass))
		req.Header.Set("Authorization", "Basic "+encodedAuth)
	}

	resp, err := rpcClient.Do(req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}
