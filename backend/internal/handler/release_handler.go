package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"sterile-packaging-release-control/backend/internal/dto"
	"sterile-packaging-release-control/backend/internal/service"
	"sterile-packaging-release-control/backend/internal/util"
)

type ReleaseHandler struct{ service service.ReleaseService }

func NewReleaseHandler(releaseService service.ReleaseService) *ReleaseHandler {
	return &ReleaseHandler{service: releaseService}
}

func (h *ReleaseHandler) List(c *gin.Context) {
	var query dto.PageQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		util.RespondError(c, util.BadRequest(err.Error()))
		return
	}
	result, err := h.service.List(c.Request.Context(), query, c.Query("decision"))
	if err != nil {
		util.RespondError(c, err)
		return
	}
	util.Respond(c, http.StatusOK, result)
}

func (h *ReleaseHandler) Get(c *gin.Context) {
	id, ok := util.ParseID(c)
	if !ok {
		return
	}
	decision, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		util.RespondError(c, err)
		return
	}
	util.Respond(c, http.StatusOK, decision)
}

func (h *ReleaseHandler) Decide(c *gin.Context) {
	var input dto.CreateReleaseDecisionRequest
	if !util.BindJSON(c, &input) {
		return
	}
	decision, err := h.service.Decide(c.Request.Context(), ActorFromContext(c), input)
	if err != nil {
		util.RespondError(c, err)
		return
	}
	util.Respond(c, http.StatusCreated, decision)
}
