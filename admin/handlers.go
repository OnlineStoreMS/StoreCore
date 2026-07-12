package admin

import (
	"net/http"
	"strconv"
	"strings"

	"storecore/internal/dto"
	"storecore/internal/integrations/productcore"
	"storecore/internal/model"
	"storecore/internal/pkg/authcontext"
	"storecore/internal/pkg/httputil"
	"storecore/internal/pkg/response"
	"storecore/internal/service"

	"github.com/gin-gonic/gin"
)

type StoreHandler struct {
	svc *service.StoreService
}

func NewStoreHandler(svc *service.StoreService) *StoreHandler {
	return &StoreHandler{svc: svc}
}

func (h *StoreHandler) ss(c *gin.Context) *service.StoreService {
	return h.svc.ForTenant(authcontext.TenantID(c))
}

func (h *StoreHandler) List(c *gin.Context) {
	page, pageSize := httputil.ParsePage(c)
	list, total, err := h.ss(c).List(c.Query("keyword"), page, pageSize)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.OK(c, response.PageResult(list, total, page, pageSize))
}

func (h *StoreHandler) Get(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	item, err := h.ss(c).Get(id)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *StoreHandler) Create(c *gin.Context) {
	var in dto.StoreDTO
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.ss(c).Create(&in)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.Created(c, item)
}

func (h *StoreHandler) Update(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	var in dto.StoreDTO
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.ss(c).Update(id, &in)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *StoreHandler) Delete(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.ss(c).Delete(id); err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, gin.H{"deleted": true})
}

type PosHandler struct {
	svc *service.PosService
}

func NewPosHandler(svc *service.PosService) *PosHandler {
	return &PosHandler{svc: svc}
}

func (h *PosHandler) ss(c *gin.Context) *service.PosService {
	return h.svc.ForTenant(authcontext.TenantID(c))
}

func (h *PosHandler) List(c *gin.Context) {
	page, pageSize := httputil.ParsePage(c)
	list, total, err := h.ss(c).List(httputil.ParseStoreID(c), page, pageSize)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.OK(c, response.PageResult(list, total, page, pageSize))
}

func (h *PosHandler) Get(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	item, err := h.ss(c).Get(id)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *PosHandler) Create(c *gin.Context) {
	var in dto.PosOrderDTO
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.ss(c).Create(&in, authcontext.UserID(c))
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.Created(c, item)
}

func (h *PosHandler) MarkPaid(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	item, err := h.ss(c).MarkPaid(id)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *PosHandler) Delete(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.ss(c).Delete(id); err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, nil)
}

type SalesHandler struct {
	svc *service.SalesService
}

func NewSalesHandler(svc *service.SalesService) *SalesHandler {
	return &SalesHandler{svc: svc}
}

func (h *SalesHandler) ss(c *gin.Context) *service.SalesService {
	return h.svc.ForTenant(authcontext.TenantID(c))
}

func (h *SalesHandler) List(c *gin.Context) {
	page, pageSize := httputil.ParsePage(c)
	list, total, err := h.ss(c).List(httputil.ParseStoreID(c), c.Query("status"), page, pageSize)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.OK(c, response.PageResult(list, total, page, pageSize))
}

func (h *SalesHandler) Get(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	item, err := h.ss(c).Get(id)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *SalesHandler) Create(c *gin.Context) {
	var in dto.StoreSalesOrderDTO
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.ss(c).Create(&in, authcontext.UserID(c))
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.Created(c, item)
}

func (h *SalesHandler) Update(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	var in dto.StoreSalesOrderDTO
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.ss(c).Update(id, &in)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *SalesHandler) Confirm(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	item, err := h.ss(c).Confirm(id)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *SalesHandler) Cancel(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	item, err := h.ss(c).Cancel(id)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *SalesHandler) Delete(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.ss(c).Delete(id); err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, nil)
}

func (h *SalesHandler) MarkReady(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	item, err := h.ss(c).MarkReady(id)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *SalesHandler) Ship(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	item, err := h.ss(c).Ship(id)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *SalesHandler) Complete(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	item, err := h.ss(c).Complete(id)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *SalesHandler) ScheduleExpress(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	var in struct {
		ScheduledAt *string `json:"scheduledAt"`
		Company     string  `json:"company"`
	}
	_ = c.ShouldBindJSON(&in)
	item, err := h.ss(c).ScheduleExpress(id, in.ScheduledAt, in.Company)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *SalesHandler) RefreshReceipt(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	preview := c.Query("preview") == "1" || c.Query("preview") == "true"
	item, err := h.ss(c).RefreshReceipt(id, preview)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, item)
}

type ServiceHandler struct {
	svc *service.ServiceOrderService
}

func NewServiceHandler(svc *service.ServiceOrderService) *ServiceHandler {
	return &ServiceHandler{svc: svc}
}

func (h *ServiceHandler) ss(c *gin.Context) *service.ServiceOrderService {
	return h.svc.ForTenant(authcontext.TenantID(c))
}

func (h *ServiceHandler) List(c *gin.Context) {
	page, pageSize := httputil.ParsePage(c)
	list, total, err := h.ss(c).List(httputil.ParseStoreID(c), page, pageSize)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.OK(c, response.PageResult(list, total, page, pageSize))
}

func (h *ServiceHandler) Create(c *gin.Context) {
	var in dto.ServiceOrderDTO
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.ss(c).Create(&in, authcontext.UserID(c))
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.Created(c, item)
}

func (h *ServiceHandler) Get(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	item, err := h.ss(c).Get(id)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *ServiceHandler) Update(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	var in dto.ServiceOrderDTO
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.ss(c).Update(id, &in)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *ServiceHandler) UpdateStatus(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	var in dto.StatusActionDTO
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.ss(c).UpdateStatus(id, in.Status)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *ServiceHandler) Delete(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.ss(c).Delete(id); err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, nil)
}

type InventoryHandler struct {
	svc *service.InventoryService
	pc  *productcore.Client
}

func NewInventoryHandler(svc *service.InventoryService, pc *productcore.Client) *InventoryHandler {
	return &InventoryHandler{svc: svc, pc: pc}
}

func (h *InventoryHandler) ss(c *gin.Context) *service.InventoryService {
	return h.svc.ForTenant(authcontext.TenantID(c))
}

func (h *InventoryHandler) List(c *gin.Context) {
	page, pageSize := httputil.ParsePage(c)
	storeID := httputil.ParseStoreID(c)
	keyword := c.Query("keyword")
	brandID, _ := strconv.ParseUint(c.Query("brandId"), 10, 64)
	categoryID, _ := strconv.ParseUint(c.Query("categoryId"), 10, 64)
	groupID, _ := strconv.ParseUint(c.Query("groupId"), 10, 64)

	var (
		list  []model.StoreInventory
		total int64
		err   error
	)
	if brandID > 0 || categoryID > 0 || groupID > 0 {
		skuIDs, cerr := h.pc.CollectSkuIDs(c.Request.Context(), c.GetHeader("Authorization"), brandID, categoryID, groupID)
		if cerr != nil {
			response.Fail(c, http.StatusBadGateway, cerr.Error())
			return
		}
		list, total, err = h.ss(c).ListBySkuIDs(storeID, skuIDs, keyword, page, pageSize)
	} else {
		list, total, err = h.ss(c).List(storeID, keyword, page, pageSize)
	}
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.OK(c, response.PageResult(list, total, page, pageSize))
}

func (h *InventoryHandler) Adjust(c *gin.Context) {
	var in dto.InventoryAdjustDTO
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.ss(c).Adjust(&in)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *InventoryHandler) BySkus(c *gin.Context) {
	storeID := httputil.ParseStoreID(c)
	if storeID == 0 {
		response.Fail(c, http.StatusBadRequest, "storeId required")
		return
	}
	raw := strings.TrimSpace(c.Query("skuIds"))
	var skuIDs []uint64
	if raw != "" {
		for _, part := range strings.Split(raw, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			id, err := strconv.ParseUint(part, 10, 64)
			if err != nil {
				response.Fail(c, http.StatusBadRequest, "invalid skuIds")
				return
			}
			skuIDs = append(skuIDs, id)
		}
	}
	qtyMap, err := h.ss(c).MapQtyBySkuIDs(storeID, skuIDs)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	// 转为数组便于前端：[{skuId, quantity}]
	type row struct {
		SkuID    uint64 `json:"skuId"`
		Quantity int    `json:"quantity"`
	}
	list := make([]row, 0, len(qtyMap))
	for id, qty := range qtyMap {
		list = append(list, row{SkuID: id, Quantity: qty})
	}
	response.OK(c, list)
}

func (h *InventoryHandler) ListByStore(c *gin.Context) {
	storeID := httputil.ParseStoreID(c)
	list, err := h.ss(c).ListByStore(storeID)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, list)
}

type StockTransferHandler struct {
	svc *service.StockTransferService
}

func NewStockTransferHandler(svc *service.StockTransferService) *StockTransferHandler {
	return &StockTransferHandler{svc: svc}
}

func (h *StockTransferHandler) ss(c *gin.Context) *service.StockTransferService {
	return h.svc.ForTenant(authcontext.TenantID(c))
}

func (h *StockTransferHandler) List(c *gin.Context) {
	page, pageSize := httputil.ParsePage(c)
	list, total, err := h.ss(c).List(httputil.ParseStoreID(c), page, pageSize)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.OK(c, response.PageResult(list, total, page, pageSize))
}

func (h *StockTransferHandler) Get(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	item, err := h.ss(c).Get(id)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *StockTransferHandler) Create(c *gin.Context) {
	var in dto.StockTransferOrderDTO
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.ss(c).Create(&in, authcontext.UserID(c))
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.Created(c, item)
}

func (h *StockTransferHandler) Confirm(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	item, err := h.ss(c).Confirm(id)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *StockTransferHandler) Cancel(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	item, err := h.ss(c).Cancel(id)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, item)
}

type PurchaseHandler struct {
	svc *service.PurchaseService
}

func NewPurchaseHandler(svc *service.PurchaseService) *PurchaseHandler {
	return &PurchaseHandler{svc: svc}
}

func (h *PurchaseHandler) ss(c *gin.Context) *service.PurchaseService {
	return h.svc.ForTenant(authcontext.TenantID(c))
}

func (h *PurchaseHandler) List(c *gin.Context) {
	page, pageSize := httputil.ParsePage(c)
	list, total, err := h.ss(c).List(httputil.ParseStoreID(c), page, pageSize)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.OK(c, response.PageResult(list, total, page, pageSize))
}

func (h *PurchaseHandler) Create(c *gin.Context) {
	var in dto.StorePurchaseOrderDTO
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.ss(c).Create(&in, authcontext.UserID(c))
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.Created(c, item)
}

func (h *PurchaseHandler) Get(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	item, err := h.ss(c).Get(id)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *PurchaseHandler) Submit(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	item, err := h.ss(c).Submit(id)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *PurchaseHandler) Receive(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	item, err := h.ss(c).Receive(id)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *PurchaseHandler) Cancel(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	item, err := h.ss(c).Cancel(id)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *PurchaseHandler) CreateFromSales(c *gin.Context) {
	salesID, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid sales order id")
		return
	}
	var in dto.StorePurchaseOrderDTO
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.ss(c).CreateFromSales(salesID, &in, authcontext.UserID(c))
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.Created(c, item)
}

type SurveillanceHandler struct {
	svc *service.SurveillanceService
}

func NewSurveillanceHandler(svc *service.SurveillanceService) *SurveillanceHandler {
	return &SurveillanceHandler{svc: svc}
}

func (h *SurveillanceHandler) ss(c *gin.Context) *service.SurveillanceService {
	return h.svc.ForTenant(authcontext.TenantID(c))
}

func (h *SurveillanceHandler) List(c *gin.Context) {
	page, pageSize := httputil.ParsePage(c)
	list, total, err := h.ss(c).List(httputil.ParseStoreID(c), page, pageSize)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.OK(c, response.PageResult(list, total, page, pageSize))
}

func (h *SurveillanceHandler) Create(c *gin.Context) {
	var in dto.SurveillanceDeviceDTO
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.ss(c).Create(&in)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.Created(c, item)
}

type ProductSkuHandler struct {
	pc *productcore.Client
}

func NewProductSkuHandler(pc *productcore.Client) *ProductSkuHandler {
	return &ProductSkuHandler{pc: pc}
}

func (h *ProductSkuHandler) Search(c *gin.Context) {
	keyword := c.Query("keyword")
	page, pageSize := httputil.ParsePage(c)
	list, total, err := h.pc.SearchSkus(c.Request.Context(), c.GetHeader("Authorization"), keyword, page, pageSize)
	if err != nil {
		response.Fail(c, http.StatusBadGateway, err.Error())
		return
	}
	response.OK(c, response.PageResult(list, total, page, pageSize))
}

func (h *ProductSkuHandler) CategoryTree(c *gin.Context) {
	tree, err := h.pc.GetCategoryTree(c.Request.Context(), c.GetHeader("Authorization"))
	if err != nil {
		response.Fail(c, http.StatusBadGateway, err.Error())
		return
	}
	response.OK(c, tree)
}

func (h *ProductSkuHandler) ListProducts(c *gin.Context) {
	keyword := c.Query("keyword")
	categoryID, _ := strconv.ParseUint(c.Query("categoryId"), 10, 64)
	brandID, _ := strconv.ParseUint(c.Query("brandId"), 10, 64)
	groupID, _ := strconv.ParseUint(c.Query("groupId"), 10, 64)
	page, pageSize := httputil.ParsePage(c)
	list, total, err := h.pc.ListProducts(c.Request.Context(), c.GetHeader("Authorization"), keyword, categoryID, brandID, groupID, page, pageSize)
	if err != nil {
		response.Fail(c, http.StatusBadGateway, err.Error())
		return
	}
	response.OK(c, response.PageResult(list, total, page, pageSize))
}

func (h *ProductSkuHandler) ListBrands(c *gin.Context) {
	list, err := h.pc.ListBrands(c.Request.Context(), c.GetHeader("Authorization"))
	if err != nil {
		response.Fail(c, http.StatusBadGateway, err.Error())
		return
	}
	response.OK(c, list)
}

func (h *ProductSkuHandler) ListGroups(c *gin.Context) {
	list, err := h.pc.ListGroups(c.Request.Context(), c.GetHeader("Authorization"))
	if err != nil {
		response.Fail(c, http.StatusBadGateway, err.Error())
		return
	}
	response.OK(c, list)
}

func (h *ProductSkuHandler) GetProductSkus(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	item, err := h.pc.GetProductSkus(c.Request.Context(), c.GetHeader("Authorization"), id)
	if err != nil {
		response.Fail(c, http.StatusBadGateway, err.Error())
		return
	}
	response.OK(c, item)
}

type ReceiptTemplateHandler struct {
	svc *service.ReceiptTemplateService
}

func NewReceiptTemplateHandler(svc *service.ReceiptTemplateService) *ReceiptTemplateHandler {
	return &ReceiptTemplateHandler{svc: svc}
}

func (h *ReceiptTemplateHandler) ss(c *gin.Context) *service.ReceiptTemplateService {
	return h.svc.ForTenant(authcontext.TenantID(c))
}

func (h *ReceiptTemplateHandler) List(c *gin.Context) {
	page, pageSize := httputil.ParsePage(c)
	list, total, err := h.ss(c).List(httputil.ParseStoreID(c), c.Query("receiptType"), page, pageSize)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.OK(c, response.PageResult(list, total, page, pageSize))
}

func (h *ReceiptTemplateHandler) Get(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	item, err := h.ss(c).Get(id)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *ReceiptTemplateHandler) Create(c *gin.Context) {
	var in dto.ReceiptTemplateDTO
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.ss(c).Create(&in)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.Created(c, item)
}

func (h *ReceiptTemplateHandler) Update(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	var in dto.ReceiptTemplateDTO
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.ss(c).Update(id, &in)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *ReceiptTemplateHandler) Delete(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.ss(c).Delete(id); err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, nil)
}

type ServiceCatalogHandler struct {
	svc *service.ServiceCatalogService
}

func NewServiceCatalogHandler(svc *service.ServiceCatalogService) *ServiceCatalogHandler {
	return &ServiceCatalogHandler{svc: svc}
}

func (h *ServiceCatalogHandler) ss(c *gin.Context) *service.ServiceCatalogService {
	return h.svc.ForTenant(authcontext.TenantID(c))
}

func (h *ServiceCatalogHandler) CategoryTree(c *gin.Context) {
	tree, err := h.ss(c).CategoryTree()
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.OK(c, tree)
}

func (h *ServiceCatalogHandler) CreateCategory(c *gin.Context) {
	var in dto.ServiceCategoryDTO
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.ss(c).CreateCategory(&in)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.Created(c, item)
}

func (h *ServiceCatalogHandler) UpdateCategory(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	var in dto.ServiceCategoryDTO
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.ss(c).UpdateCategory(id, &in)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *ServiceCatalogHandler) DeleteCategory(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.ss(c).DeleteCategory(id); err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, nil)
}

func (h *ServiceCatalogHandler) ListItems(c *gin.Context) {
	page, pageSize := httputil.ParsePage(c)
	categoryID, _ := strconv.ParseUint(c.Query("categoryId"), 10, 64)
	var statusPtr *int8
	if s := c.Query("status"); s != "" {
		v, err := strconv.ParseInt(s, 10, 8)
		if err == nil {
			st := int8(v)
			statusPtr = &st
		}
	}
	list, total, err := h.ss(c).ListItems(categoryID, c.Query("keyword"), statusPtr, page, pageSize)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.OK(c, response.PageResult(list, total, page, pageSize))
}

func (h *ServiceCatalogHandler) GetItem(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	item, err := h.ss(c).GetItem(id)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *ServiceCatalogHandler) CreateItem(c *gin.Context) {
	var in dto.ServiceItemDTO
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.ss(c).CreateItem(&in)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.Created(c, item)
}

func (h *ServiceCatalogHandler) UpdateItem(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	var in dto.ServiceItemDTO
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.ss(c).UpdateItem(id, &in)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *ServiceCatalogHandler) DeleteItem(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.ss(c).DeleteItem(id); err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, nil)
}
