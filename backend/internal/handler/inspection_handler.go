package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"sterile-packaging-release-control/backend/internal/dto"
	"sterile-packaging-release-control/backend/internal/repository"
	"sterile-packaging-release-control/backend/internal/service"
	"sterile-packaging-release-control/backend/internal/util"
)

type InspectionHandler struct{ service service.InspectionService }

func NewInspectionHandler(inspectionService service.InspectionService) *InspectionHandler {
	return &InspectionHandler{service: inspectionService}
}

func (h *InspectionHandler) List(c *gin.Context) {
	var filter repository.InspectionFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		util.RespondError(c, util.BadRequest(err.Error()))
		return
	}
	result, err := h.service.List(c.Request.Context(), filter)
	if err != nil {
		util.RespondError(c, err)
		return
	}
	util.Respond(c, http.StatusOK, result)
}

func (h *InspectionHandler) Get(c *gin.Context) {
	id, ok := util.ParseID(c)
	if !ok {
		return
	}
	sample, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		util.RespondError(c, err)
		return
	}
	util.Respond(c, http.StatusOK, sample)
}

func (h *InspectionHandler) Create(c *gin.Context) {
	var input dto.CreateInspectionRequest
	if !util.BindJSON(c, &input) {
		return
	}
	sample, err := h.service.Create(c.Request.Context(), ActorFromContext(c), input)
	if err != nil {
		util.RespondError(c, err)
		return
	}
	util.Respond(c, http.StatusCreated, sample)
}

func (h *InspectionHandler) Complete(c *gin.Context) {
	id, ok := util.ParseID(c)
	if !ok {
		return
	}
	var input dto.CompleteInspectionRequest
	if !util.BindJSON(c, &input) {
		return
	}
	sample, err := h.service.Complete(c.Request.Context(), ActorFromContext(c), id, input)
	if err != nil {
		util.RespondError(c, err)
		return
	}
	util.Respond(c, http.StatusOK, sample)
}

func (h *InspectionHandler) Retest(c *gin.Context) {
	id, ok := util.ParseID(c)
	if !ok {
		return
	}
	var input struct {
		Reason string `json:"reason" binding:"required,min=3,max=500"`
	}
	if !util.BindJSON(c, &input) {
		return
	}
	sample, err := h.service.RequestRetest(c.Request.Context(), ActorFromContext(c), id, input.Reason)
	if err != nil {
		util.RespondError(c, err)
		return
	}
	util.Respond(c, http.StatusOK, sample)
}
