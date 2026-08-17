package health

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

type Checker struct {
	host     string
	port     string
	services []string
	client   HTTPDoer
	logger   Logger
}

type HTTPDoer interface {
	Do(*http.Request) (*http.Response, error)
}

type Logger interface {
	Infof(string, ...interface{})
}

type standardLogger struct{}

func (standardLogger) Infof(format string, args ...interface{}) { log.Printf(format, args...) }

type ServiceStatusRequest struct {
	Service string `json:"service"`
}

type ServiceStatusResponse struct {
	Service string `json:"service"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

func New() *Checker {
	return NewWithDependencies(http.DefaultClient, standardLogger{})
}

func NewWithDependencies(client HTTPDoer, logger Logger) *Checker {
	return &Checker{client: client, logger: logger}
}

func (c *Checker) SetHost(host string) *Checker {
	c.host = host
	return c
}

func (c *Checker) SetPort(port string) *Checker {
	c.port = port
	return c
}

func (c *Checker) SetServices(services []string) *Checker {
	c.services = services
	return c
}

func (c *Checker) GetHost() string {
	return c.host
}

func (c *Checker) GetPort() string {
	return c.port
}

func (c *Checker) Run() {
	for _, service := range c.services {
		c.checkService(c.client, service)
	}
}

func (c *Checker) checkService(client HTTPDoer, serviceName string) {
	url := fmt.Sprintf("http://%s:%s/akoflow-server/check-service", c.GetHost(), c.GetPort())

	payload := ServiceStatusRequest{
		Service: serviceName,
	}

	payloadJson, _ := json.Marshal(payload) // a string-only DTO cannot fail JSON encoding

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payloadJson))
	if err != nil {
		c.logger.Infof("Service %s Failed: %v\n", serviceName, err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		c.logger.Infof("Service %s Failed: %v\n", serviceName, err)
		return
	}
	defer resp.Body.Close()

	var result ServiceStatusResponse
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		c.logger.Infof("Service %s Failed: %v\n", serviceName, err)
		return
	}

	if result.Status == "OK" {
		c.logger.Infof("Service %s OK\n", serviceName)
	} else {
		c.logger.Infof("Service %s Failed: %s\n", serviceName, result.Message)
	}
}
