package broker

import (
	"log/slog"
	"net/http"
	"sync"

	"github.com/gogolok/osb-broker-lib/pkg/broker"

	"reflect"

	osb "sigs.k8s.io/go-open-service-broker-client/v2"
)

// NewBusinessLogic is a hook that is called with the Options the program is run
// with. NewBusinessLogic is the place where you will initialize your
// BusinessLogic the parameters passed in.
func NewBusinessLogic(o Options) (*BusinessLogic, error) {
	b := BusinessLogic{
		async:     true,
		instances: make(map[string]*exampleInstance, 10),
	}

	// HACK: Enable if you have existing instances.
	instanceId := "9ca5dd24-5010-4c7f-89e9-71f67d35fd6d"
	exampleInstance := &exampleInstance{
		ID:        instanceId,
		ServiceID: "serviceId",
		PlanID:    "planId",
		Params:    map[string]interface{}{},
	}
	b.instances[instanceId] = exampleInstance

	// For example, if your BusinessLogic requires a parameter from the command
	// line, you would unpack it from the Options and set it on the
	// BusinessLogic here.
	return &b, nil
}

// BusinessLogic provides an implementation of the broker.BusinessLogic
// interface.
type BusinessLogic struct {
	// Indicates if the broker should handle the requests asynchronously.
	async bool
	// Synchronize go routines.
	sync.RWMutex
	// Add fields here! These fields are provided purely as an example
	instances map[string]*exampleInstance
}

var _ broker.Interface = &BusinessLogic{}

func truePtr() *bool {
	b := true
	return &b
}

func (b *BusinessLogic) GetCatalog(c *broker.RequestContext) (*broker.CatalogResponse, error) {
	// Your catalog business logic goes here
	response := &broker.CatalogResponse{}
	osbResponse := &osb.CatalogResponse{
		Services: []osb.Service{
			{
				Name:          "example-broker",
				ID:            "4f6e6cf6-33dd-425f-a2c7-3c9258ad246a",
				Description:   "An example service",
				Bindable:      true,
				PlanUpdatable: truePtr(),
				Metadata: map[string]interface{}{
					"displayName": "Example service",
					"imageUrl":    "https://avatars2.githubusercontent.com/u/19862012?s=200&v=4",
				},
				Plans: []osb.Plan{
					{
						Name:        "default",
						ID:          "86064792-7ea2-467b-af93-ac9694d96d5b",
						Description: "The default plan for the example service",
						Free:        truePtr(),
						Metadata: map[string]interface{}{
							"hello": "world",
						},
						Schemas: &osb.Schemas{
							ServiceInstance: &osb.ServiceInstanceSchema{
								Create: &osb.InputParametersSchema{
									Parameters: map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"color": map[string]interface{}{
												"type":    "string",
												"default": "Clear",
												"enum": []string{
													"Clear",
													"Beige",
													"Grey",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	slog.Info("catalog response", "osbReponse", osbResponse)

	response.CatalogResponse = *osbResponse

	return response, nil
}

func (b *BusinessLogic) Provision(request *osb.ProvisionRequest, c *broker.RequestContext) (*broker.ProvisionResponse, error) {
	// Your provision business logic goes here

	// example implementation:
	b.Lock()
	defer b.Unlock()

	response := broker.ProvisionResponse{}

	exampleInstance := &exampleInstance{
		ID:        request.InstanceID,
		ServiceID: request.ServiceID,
		PlanID:    request.PlanID,
		Params:    request.Parameters,
	}
	slog.Info("New example instance", "instance", exampleInstance)

	// Check to see if this is the same instance
	if i := b.instances[request.InstanceID]; i != nil {
		if i.Match(exampleInstance) {
			response.Exists = true
			return &response, nil
		} else {
			// Instance ID in use, this is a conflict.
			description := "InstanceID in use"
			return nil, osb.HTTPStatusCodeError{
				StatusCode:  http.StatusConflict,
				Description: &description,
			}
		}
	}
	b.instances[request.InstanceID] = exampleInstance

	if request.AcceptsIncomplete {
		response.Async = b.async
	}

	return &response, nil
}

func (b *BusinessLogic) Deprovision(request *osb.DeprovisionRequest, c *broker.RequestContext) (*broker.DeprovisionResponse, error) {
	// Your deprovision business logic goes here

	// example implementation:
	b.Lock()
	defer b.Unlock()

	response := broker.DeprovisionResponse{}

	delete(b.instances, request.InstanceID)

	if request.AcceptsIncomplete {
		response.Async = b.async
	}

	return &response, nil
}

func (b *BusinessLogic) LastOperation(request *osb.LastOperationRequest, c *broker.RequestContext) (*broker.LastOperationResponse, error) {
	b.Lock()
	defer b.Unlock()

	_, ok := b.instances[request.InstanceID]
	if !ok {
		slog.Error("instance not found", "instance", request.InstanceID)
		return nil, osb.HTTPStatusCodeError{
			StatusCode: http.StatusNotFound,
		}
	}

	// Your last-operation business logic goes here
	lor := broker.LastOperationResponse{
		LastOperationResponse: osb.LastOperationResponse{
			State: osb.StateSucceeded,
		},
	}

	return &lor, nil
}

func (b *BusinessLogic) Bind(request *osb.BindRequest, c *broker.RequestContext) (*broker.BindResponse, error) {
	// Your bind business logic goes here

	// example implementation:
	b.Lock()
	defer b.Unlock()

	instance, ok := b.instances[request.InstanceID]
	if !ok {
		slog.Error("instance not found", "instance", request.InstanceID)
		return nil, osb.HTTPStatusCodeError{
			StatusCode: http.StatusNotFound,
		}
	}

	response := broker.BindResponse{
		BindResponse: osb.BindResponse{
			Credentials: instance.Params,
		},
	}
	if request.AcceptsIncomplete {
		response.Async = b.async
	}

	return &response, nil
}

func (b *BusinessLogic) Unbind(request *osb.UnbindRequest, c *broker.RequestContext) (*broker.UnbindResponse, error) {
	// Your unbind business logic goes here
	return &broker.UnbindResponse{}, nil
}

func (b *BusinessLogic) Update(request *osb.UpdateInstanceRequest, c *broker.RequestContext) (*broker.UpdateInstanceResponse, error) {
	// Your logic for updating a service goes here.
	response := broker.UpdateInstanceResponse{}
	if request.AcceptsIncomplete {
		response.Async = b.async
	}

	return &response, nil
}

func (b *BusinessLogic) ValidateBrokerAPIVersion(version string) error {
	return nil
}

// example types

// exampleInstance is intended as an example of a type that holds information about a service instance
type exampleInstance struct {
	ID        string
	ServiceID string
	PlanID    string
	Params    map[string]interface{}
}

func (i *exampleInstance) Match(other *exampleInstance) bool {
	return reflect.DeepEqual(i, other)
}

func (b *BusinessLogic) BindingLastOperation(request *osb.BindingLastOperationRequest, c *broker.RequestContext) (*broker.LastOperationResponse, error) {
	lor := broker.LastOperationResponse{
		LastOperationResponse: osb.LastOperationResponse{
			State: osb.StateSucceeded,
		},
	}

	return &lor, nil
}

func (b *BusinessLogic) GetBinding(request *osb.GetBindingRequest, c *broker.RequestContext) (*broker.GetBindingResponse, error) {
	response := broker.GetBindingResponse{
		GetBindingResponse: osb.GetBindingResponse{
			Credentials: map[string]interface{}{
				"uri":      "mysql://mysqluser:pass@mysqlhost:3306/dbname",
				"username": "mysqluser",
				"password": "pass",
				"host":     "mysqlhost",
				"port":     float64(3306),
				"database": "dbname",
			},
		},
	}

	return &response, nil
}
