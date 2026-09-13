package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"sterile-packaging-release-control/backend/internal/constants"
	"sterile-packaging-release-control/backend/internal/dto"
	"sterile-packaging-release-control/backend/internal/repository"
	"sterile-packaging-release-control/backend/internal/service"
	"sterile-packaging-release-control/backend/internal/util"
)

type BatchHandler struct{ service service.BatchService }

func NewBatchHandler(batchService service.BatchService) *BatchHandler {
	return &BatchHandler{service: batchService}
}

func (h *BatchHandler) List(c *gin.Context) {
	var filter repository.BatchFilter
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

func (h *BatchHandler) Overview(c *gin.Context) {
	result, err := h.service.Overview(c.Request.Context())
	if err != nil {
		util.RespondError(c, err)
		return
	}
	util.Respond(c, http.StatusOK, result)
}

func (h *BatchHandler) Get(c *gin.Context) {
	id, ok := util.ParseID(c)
	if !ok {
		return
	}
	batch, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		util.RespondError(c, err)
		return
	}
	util.Respond(c, http.StatusOK, batch)
}

func (h *BatchHandler) Create(c *gin.Context) {
	var input dto.CreateBatchRequest
	if !util.BindJSON(c, &input) {
		return
	}
	batch, err := h.service.Create(c.Request.Context(), ActorFromContext(c), input)
	if err != nil {
		util.RespondError(c, err)
		return
	}
	util.Respond(c, http.StatusCreated, batch)
}

func (h *BatchHandler) Update(c *gin.Context) {
	id, ok := util.ParseID(c)
	if !ok {
		return
	}
	var input dto.UpdateBatchRequest
	if !util.BindJSON(c, &input) {
		return
	}
	batch, err := h.service.Update(c.Request.Context(), ActorFromContext(c), id, input)
	if err != nil {
		util.RespondError(c, err)
		return
	}
	util.Respond(c, http.StatusOK, batch)
}

func (h *BatchHandler) Transition(c *gin.Context) {
	id, ok := util.ParseID(c)
	if !ok {
		return
	}
	var input dto.TransitionRequest
	if !util.BindJSON(c, &input) {
		return
	}
	batch, err := h.service.Transition(c.Request.Context(), ActorFromContext(c), id, constants.BatchStatus(input.Status), input.Reason)
	if err != nil {
		util.RespondError(c, err)
		return
	}
	util.Respond(c, http.StatusOK, batch)
}
