package admin

import (
	"net/http"
	"strconv"

	"storecore/internal/dto"
	"storecore/internal/pkg/httputil"
	"storecore/internal/pkg/response"
	"storecore/internal/repo"
	"storecore/internal/service"

	"github.com/gin-gonic/gin"
)

type InternalHandlers struct {
	store   *service.StoreService
	service *service.ServiceOrderService
	catalog *service.ServiceCatalogService
}

func NewInternalHandlers(
	store *service.StoreService,
	svc *service.ServiceOrderService,
	catalog *service.ServiceCatalogService,
) *InternalHandlers {
	return &InternalHandlers{store: store, service: svc, catalog: catalog}
}

func internalTenantID(c *gin.Context, bodyTenant uint64) uint64 {
	if bodyTenant > 0 {
		return repo.NormalizeTenantID(bodyTenant)
	}
	if v := c.GetHeader("X-Tenant-Id"); v != "" {
		if id, err := strconv.ParseUint(v, 10, 64); err == nil && id > 0 {
			return repo.NormalizeTenantID(id)
		}
	}
	if q := c.Query("tenantId"); q != "" {
		if id, err := strconv.ParseUint(q, 10, 64); err == nil && id > 0 {
			return repo.NormalizeTenantID(id)
		}
	}
	return repo.NormalizeTenantID(0)
}

func (h *InternalHandlers) ListStores(c *gin.Context) {
	page, pageSize := httputil.ParsePage(c)
	tenantID := internalTenantID(c, 0)
	list, total, err := h.store.ForTenant(tenantID).List(c.Query("keyword"), page, pageSize)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.OK(c, response.PageResult(list, total, page, pageSize))
}

func (h *InternalHandlers) ListServiceCatalog(c *gin.Context) {
	page, pageSize := httputil.ParsePage(c)
	tenantID := internalTenantID(c, 0)
	status := int8(1)
	list, total, err := h.catalog.ForTenant(tenantID).ListItems(0, c.Query("keyword"), &status, page, pageSize)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.OK(c, response.PageResult(list, total, page, pageSize))
}

func (h *InternalHandlers) CreateServiceOrder(c *gin.Context) {
	var in struct {
		dto.ServiceOrderDTO
		TenantID uint64 `json:"tenantId"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	tenantID := internalTenantID(c, in.TenantID)
	item, err := h.service.ForTenant(tenantID).Create(&in.ServiceOrderDTO, 0)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.Created(c, item)
}

func (h *InternalHandlers) ListServiceOrders(c *gin.Context) {
	page, pageSize := httputil.ParsePage(c)
	tenantID := internalTenantID(c, 0)
	keyword := c.Query("keyword")
	if phone := c.Query("customerPhone"); phone != "" {
		keyword = phone
	}
	f := dto.ServiceOrderListFilter{
		Status:    c.Query("status"),
		PayStatus: c.Query("payStatus"),
		OrderMode: c.Query("orderMode"),
		Keyword:   keyword,
	}
	storeID, _ := strconv.ParseUint(c.Query("storeId"), 10, 64)
	list, total, err := h.service.ForTenant(tenantID).List(storeID, f, page, pageSize)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.OK(c, response.PageResult(list, total, page, pageSize))
}

func RegisterInternalRoutes(g *gin.RouterGroup, h *InternalHandlers) {
	g.GET("/stores", h.ListStores)
	g.GET("/service-catalog", h.ListServiceCatalog)
	g.POST("/service-orders", h.CreateServiceOrder)
	g.GET("/service-orders", h.ListServiceOrders)
}
