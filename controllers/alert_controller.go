package controllers

import (
	"net/http"
	"strconv"

	"docklog/models"
	"docklog/services"

	"github.com/labstack/echo/v4"
)

type AlertController struct {
	engine        *services.AlertEngine
	auditLogger   AuditLogger
	sessionResolver SessionResolver
}

func NewAlertController(engine *services.AlertEngine, audit AuditLogger, session SessionResolver) *AlertController {
	return &AlertController{engine: engine, auditLogger: audit, sessionResolver: session}
}

func (ac *AlertController) RegisterRoutes(admin *echo.Group) {
	admin.GET("/alerts", ac.List)
	admin.POST("/alerts", ac.Create)
	admin.PUT("/alerts/:id", ac.Update)
	admin.DELETE("/alerts/:id", ac.Delete)
	admin.POST("/alerts/templates/:ruleKey", ac.CreateFromTemplate)
	admin.POST("/alerts/test", ac.Test)
	admin.GET("/alerts/history", ac.History)
	admin.GET("/alerts/history/:id/deliveries", ac.Deliveries)
}

func (ac *AlertController) List(c echo.Context) error {
	response, err := ac.engine.GetPublic(0)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to load alert rules"})
	}
	response.History = nil
	return c.JSON(http.StatusOK, response)
}

func (ac *AlertController) History(c echo.Context) error {
	limit := 100
	if raw := c.QueryParam("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			limit = parsed
		}
	}
	response, err := ac.engine.GetPublic(limit)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to load alert history"})
	}
	return c.JSON(http.StatusOK, map[string]interface{}{
		"history": response.History,
	})
}

func (ac *AlertController) Create(c echo.Context) error {
	user, err := ac.sessionResolver(c)
	if err != nil {
		return err
	}
	var input models.AlertRuleUpsert
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}
	rule, err := ac.engine.UpsertRule(input, 0)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	ac.auditLogger(user.ID, user.Username, "CREATE_ALERT_RULE", input.RuleKey, "Success", "Alert rule created")
	return c.JSON(http.StatusCreated, rule)
}

func (ac *AlertController) Update(c echo.Context) error {
	user, err := ac.sessionResolver(c)
	if err != nil {
		return err
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid rule id"})
	}
	var input models.AlertRuleUpsert
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}
	rule, err := ac.engine.UpsertRule(input, id)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	ac.auditLogger(user.ID, user.Username, "UPDATE_ALERT_RULE", input.RuleKey, "Success", "Alert rule updated")
	return c.JSON(http.StatusOK, rule)
}

func (ac *AlertController) Delete(c echo.Context) error {
	user, err := ac.sessionResolver(c)
	if err != nil {
		return err
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid rule id"})
	}
	rule, err := ac.engine.GetRuleByID(id)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Alert rule not found"})
	}
	if err := ac.engine.DeleteRule(id); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to delete alert rule"})
	}
	ac.auditLogger(user.ID, user.Username, "DELETE_ALERT_RULE", rule.RuleKey, "Success", "Alert rule deleted")
	return c.NoContent(http.StatusNoContent)
}

func (ac *AlertController) CreateFromTemplate(c echo.Context) error {
	user, err := ac.sessionResolver(c)
	if err != nil {
		return err
	}
	var body struct {
		ChannelIDs []int64 `json:"channel_ids"`
	}
	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}
	rule, err := ac.engine.CreateFromTemplate(c.Param("ruleKey"), body.ChannelIDs)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	ac.auditLogger(user.ID, user.Username, "CREATE_ALERT_RULE", rule.RuleKey, "Success", "Alert rule created from template")
	return c.JSON(http.StatusCreated, rule)
}

func (ac *AlertController) Test(c echo.Context) error {
	var req models.AlertTestRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}
	if err := ac.engine.TestRule(req.RuleID); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]string{"message": "Test alert sent"})
}

func (ac *AlertController) Deliveries(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid history id"})
	}
	deliveries, err := ac.engine.ListDeliveries(id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to load deliveries"})
	}
	return c.JSON(http.StatusOK, deliveries)
}
