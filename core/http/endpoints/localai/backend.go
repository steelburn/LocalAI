package localai

import (
	"encoding/json"
	"fmt"
	"slices"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/mudler/LocalAI/core/config"
	"github.com/mudler/LocalAI/core/http/utils"
	"github.com/mudler/LocalAI/core/schema"
	"github.com/mudler/LocalAI/core/services"
	"github.com/rs/zerolog/log"
)

type BackendEndpointService struct {
	backends       []config.Backend
	modelPath      string
	backendApplier *services.BackendService
}

type BackendModel struct {
	ID        string `json:"id"`
	ConfigURL string `json:"config_url"`
	config.Backend
}

func CreateBackendEndpointService(backends []config.Backend, modelPath string, backendApplier *services.BackendService) BackendEndpointService {
	return BackendEndpointService{
		backends:       backends,
		modelPath:      modelPath,
		backendApplier: backendApplier,
	}
}

// GetOpStatusEndpoint returns the job status
// @Summary Returns the job status
// @Success 200 {object} services.BackendOpStatus "Response"
// @Router /backends/jobs/{uuid} [get]
func (mgs *BackendEndpointService) GetOpStatusEndpoint() func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		status := mgs.backendApplier.GetStatus(c.Params("uuid"))
		if status == nil {
			return fmt.Errorf("could not find any status for ID")
		}
		return c.JSON(status)
	}
}

// GetAllStatusEndpoint returns all the jobs status progress
// @Summary Returns all the jobs status progress
// @Success 200 {object} map[string]services.BackendOpStatus "Response"
// @Router /backends/jobs [get]
func (mgs *BackendEndpointService) GetAllStatusEndpoint() func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		return c.JSON(mgs.backendApplier.GetAllStatus())
	}
}

// ApplyBackendEndpoint installs a new backend to a LocalAI instance
// @Summary Install backends to LocalAI.
// @Param request body BackendModel true "query params"
// @Success 200 {object} schema.BackendResponse "Response"
// @Router /backends/apply [post]
func (mgs *BackendEndpointService) ApplyBackendEndpoint() func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		input := new(BackendModel)
		// Get input data from the request body
		if err := c.BodyParser(input); err != nil {
			return err
		}

		uuid, err := uuid.NewUUID()
		if err != nil {
			return err
		}
		mgs.backendApplier.C <- services.BackendOp{
			Req:       input.Backend,
			Id:        uuid.String(),
			BackendID: input.ID,
			ConfigURL: input.ConfigURL,
		}

		return c.JSON(schema.BackendResponse{ID: uuid.String(), StatusURL: fmt.Sprintf("%sbackends/jobs/%s", utils.BaseURL(c), uuid.String())})
	}
}

// DeleteBackendEndpoint lets delete backends from a LocalAI instance
// @Summary delete backends from LocalAI.
// @Param name	path string	true	"Backend name"
// @Success 200 {object} schema.BackendResponse "Response"
// @Router /backends/delete/{name} [post]
func (mgs *BackendEndpointService) DeleteBackendEndpoint() func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		backendName := c.Params("name")

		mgs.backendApplier.C <- services.BackendOp{
			Delete:    true,
			BackendID: backendName,
		}

		uuid, err := uuid.NewUUID()
		if err != nil {
			return err
		}

		return c.JSON(schema.BackendResponse{ID: uuid.String(), StatusURL: fmt.Sprintf("%sbackends/jobs/%s", utils.BaseURL(c), uuid.String())})
	}
}

// ListBackendsEndpoint list the available backends configured in LocalAI
// @Summary List all Backends
// @Success 200 {object} []config.Backend "Response"
// @Router /backends [get]
func (mgs *BackendEndpointService) ListBackendsEndpoint() func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		log.Debug().Msgf("Listing backends %+v", mgs.backends)
		dat, err := json.Marshal(mgs.backends)
		if err != nil {
			return err
		}
		return c.Send(dat)
	}
}

// AddBackendEndpoint adds a backend in LocalAI
// @Summary Adds a backend in LocalAI
// @Param request body config.Backend true "Backend details"
// @Success 200 {object} []config.Backend "Response"
// @Router /backends [post]
func (mgs *BackendEndpointService) AddBackendEndpoint() func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		input := new(config.Backend)
		// Get input data from the request body
		if err := c.BodyParser(input); err != nil {
			return err
		}
		if slices.ContainsFunc(mgs.backends, func(backend config.Backend) bool {
			return backend.Name == input.Name
		}) {
			return fmt.Errorf("%s already exists", input.Name)
		}
		dat, err := json.Marshal(mgs.backends)
		if err != nil {
			return err
		}
		log.Debug().Msgf("Adding %+v to backend list", *input)
		mgs.backends = append(mgs.backends, *input)
		return c.Send(dat)
	}
}

// RemoveBackendEndpoint remove a backend from LocalAI
// @Summary removes a backend from LocalAI
// @Param request body config.Backend true "Backend details"
// @Success 200 {object} []config.Backend "Response"
// @Router /backends [delete]
func (mgs *BackendEndpointService) RemoveBackendEndpoint() func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		input := new(config.Backend)
		// Get input data from the request body
		if err := c.BodyParser(input); err != nil {
			return err
		}
		if !slices.ContainsFunc(mgs.backends, func(backend config.Backend) bool {
			return backend.Name == input.Name
		}) {
			return fmt.Errorf("%s is not currently registered", input.Name)
		}
		mgs.backends = slices.DeleteFunc(mgs.backends, func(backend config.Backend) bool {
			return backend.Name == input.Name
		})
		dat, err := json.Marshal(mgs.backends)
		if err != nil {
			return err
		}
		return c.Send(dat)
	}
}
