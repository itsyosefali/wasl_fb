package actionmod

import (
	"github.com/gofiber/fiber/v2"
	"github.com/pop/erp_meta/internal/actions"
	"github.com/pop/erp_meta/internal/auth"
)

type Handler struct {
	executor *actions.Executor
}

func NewHandler(executor *actions.Executor) *Handler {
	return &Handler{executor: executor}
}

func (h *Handler) Register(r fiber.Router) {
	r.Post("/execute", h.Execute)
}

func (h *Handler) Execute(c *fiber.Ctx) error {
	tenant, err := auth.GetTenant(c)
	if err != nil {
		return err
	}
	var req actions.Request
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}
	if req.Action == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "action is required"})
	}
	if req.Channel == "" {
		req.Channel = "facebook"
	}

	result, err := h.executor.Execute(c.Context(), tenant.ID, req)
	if err != nil {
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(result)
}
