package admin

import (
	"net/http"

	"storecore/internal/integrations/supplycore"
	"storecore/internal/pkg/httputil"
	"storecore/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

type SupplierHandler struct {
	sc *supplycore.Client
}

func NewSupplierHandler(sc *supplycore.Client) *SupplierHandler {
	return &SupplierHandler{sc: sc}
}

func (h *SupplierHandler) List(c *gin.Context) {
	page, pageSize := httputil.ParsePage(c)
	list, total, err := h.sc.ListSuppliers(c.Request.Context(), c.GetHeader("Authorization"), c.Query("keyword"), page, pageSize)
	if err != nil {
		response.Fail(c, http.StatusBadGateway, err.Error())
		return
	}
	response.OK(c, response.PageResult(list, total, page, pageSize))
}
