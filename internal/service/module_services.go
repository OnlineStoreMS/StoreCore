package service

import (
	"errors"
	"time"

	"storecore/internal/dto"
	"storecore/internal/model"
	"storecore/internal/repo"

	"gorm.io/gorm"
)

type SalesService struct {
	repos    *repo.Repos
	tenantID uint64
}

func NewSalesService(repos *repo.Repos) *SalesService {
	return &SalesService{repos: repos}
}

func (s *SalesService) ForTenant(tenantID uint64) *SalesService {
	return &SalesService{repos: s.repos, tenantID: repo.NormalizeTenantID(tenantID)}
}

func (s *SalesService) List(storeID uint64, status string, page, pageSize int) ([]model.StoreSalesOrder, int64, error) {
	return s.repos.Sales.ForTenant(s.tenantID).List(storeID, status, page, pageSize)
}

func (s *SalesService) Get(id uint64) (*model.StoreSalesOrder, error) {
	item, err := s.repos.Sales.ForTenant(s.tenantID).GetByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return item, err
}

func (s *SalesService) Create(in *dto.StoreSalesOrderDTO, userID uint64) (*model.StoreSalesOrder, error) {
	if in.StoreID == 0 || len(in.Items) == 0 {
		return nil, ErrBadRequest
	}
	items, total := buildSalesItems(in.Items)
	ft := in.FulfillmentType
	if ft == "" {
		ft = "pickup"
	}
	order := &model.StoreSalesOrder{
		StoreID: in.StoreID,
		OrderNo: genOrderNo("SO"),
		OrderType: "offline",
		Status: "draft",
		FulfillmentType: ft,
		CustomerName: in.CustomerName,
		CustomerPhone: in.CustomerPhone,
		ShippingAddress: in.ShippingAddress,
		TotalAmount: total,
		PayStatus: "unpaid",
		NeedProcurement: in.NeedProcurement,
		Remark: in.Remark,
		CreatedBy: userID,
	}
	if err := s.repos.Sales.ForTenant(s.tenantID).Create(order, items); err != nil {
		return nil, err
	}
	return order, nil
}

type ServiceOrderService struct {
	repos    *repo.Repos
	tenantID uint64
}

func NewServiceOrderService(repos *repo.Repos) *ServiceOrderService {
	return &ServiceOrderService{repos: repos}
}

func (s *ServiceOrderService) ForTenant(tenantID uint64) *ServiceOrderService {
	return &ServiceOrderService{repos: s.repos, tenantID: repo.NormalizeTenantID(tenantID)}
}

func (s *ServiceOrderService) List(storeID uint64, page, pageSize int) ([]model.ServiceOrder, int64, error) {
	return s.repos.Service.ForTenant(s.tenantID).List(storeID, page, pageSize)
}

func (s *ServiceOrderService) Create(in *dto.ServiceOrderDTO, userID uint64) (*model.ServiceOrder, error) {
	item := &model.ServiceOrder{
		StoreID: in.StoreID,
		OrderNo: genOrderNo("SRV"),
		ServiceType: in.ServiceType,
		Status: "pending",
		CustomerName: in.CustomerName,
		CustomerPhone: in.CustomerPhone,
		DeviceInfo: in.DeviceInfo,
		FaultDesc: in.FaultDesc,
		EngineerName: in.EngineerName,
		EstimatedAmount: in.EstimatedAmount,
		Remark: in.Remark,
		CreatedBy: userID,
	}
	if in.AppointmentAt != nil && *in.AppointmentAt != "" {
		t, err := time.Parse(time.RFC3339, *in.AppointmentAt)
		if err == nil {
			item.AppointmentAt = &t
		}
	}
	if err := s.repos.Service.ForTenant(s.tenantID).Create(item); err != nil {
		return nil, err
	}
	return item, nil
}

type InventoryService struct {
	repos    *repo.Repos
	tenantID uint64
}

func NewInventoryService(repos *repo.Repos) *InventoryService {
	return &InventoryService{repos: repos}
}

func (s *InventoryService) ForTenant(tenantID uint64) *InventoryService {
	return &InventoryService{repos: s.repos, tenantID: repo.NormalizeTenantID(tenantID)}
}

func (s *InventoryService) List(storeID uint64, keyword string, page, pageSize int) ([]model.StoreInventory, int64, error) {
	return s.repos.Inventory.ForTenant(s.tenantID).List(storeID, keyword, page, pageSize)
}

func (s *InventoryService) Adjust(in *dto.InventoryAdjustDTO) (*model.StoreInventory, error) {
	item := &model.StoreInventory{
		StoreID: in.StoreID, SkuID: in.SkuID, SkuCode: in.SkuCode,
		ProductName: in.ProductName, SpecLabel: in.SpecLabel,
		Quantity: in.Quantity, SafetyStock: in.SafetyStock,
	}
	if err := s.repos.Inventory.ForTenant(s.tenantID).Upsert(item); err != nil {
		return nil, err
	}
	return item, nil
}

type PurchaseService struct {
	repos    *repo.Repos
	tenantID uint64
}

func NewPurchaseService(repos *repo.Repos) *PurchaseService {
	return &PurchaseService{repos: repos}
}

func (s *PurchaseService) ForTenant(tenantID uint64) *PurchaseService {
	return &PurchaseService{repos: s.repos, tenantID: repo.NormalizeTenantID(tenantID)}
}

func (s *PurchaseService) List(storeID uint64, page, pageSize int) ([]model.StorePurchaseOrder, int64, error) {
	return s.repos.Purchase.ForTenant(s.tenantID).List(storeID, page, pageSize)
}

func (s *PurchaseService) Create(in *dto.StorePurchaseOrderDTO, userID uint64) (*model.StorePurchaseOrder, error) {
	if in.StoreID == 0 || len(in.Items) == 0 {
		return nil, ErrBadRequest
	}
	items, total := buildPurchaseItems(in.Items)
	pt := in.PurchaseType
	if pt == "" {
		pt = "stock"
	}
	order := &model.StorePurchaseOrder{
		StoreID: in.StoreID,
		PoNo: genOrderNo("PO"),
		PurchaseType: pt,
		SupplierID: in.SupplierID,
		SupplierName: in.SupplierName,
		RefSalesID: in.RefSalesID,
		Status: "draft",
		TotalAmount: total,
		Remark: in.Remark,
		CreatedBy: userID,
	}
	if err := s.repos.Purchase.ForTenant(s.tenantID).Create(order, items); err != nil {
		return nil, err
	}
	return order, nil
}

type SurveillanceService struct {
	repos    *repo.Repos
	tenantID uint64
}

func NewSurveillanceService(repos *repo.Repos) *SurveillanceService {
	return &SurveillanceService{repos: repos}
}

func (s *SurveillanceService) ForTenant(tenantID uint64) *SurveillanceService {
	return &SurveillanceService{repos: s.repos, tenantID: repo.NormalizeTenantID(tenantID)}
}

func (s *SurveillanceService) List(storeID uint64, page, pageSize int) ([]model.SurveillanceDevice, int64, error) {
	return s.repos.Surveillance.ForTenant(s.tenantID).List(storeID, page, pageSize)
}

func (s *SurveillanceService) Create(in *dto.SurveillanceDeviceDTO) (*model.SurveillanceDevice, error) {
	item := &model.SurveillanceDevice{
		StoreID: in.StoreID, Name: in.Name, Location: in.Location,
		DeviceType: defaultDeviceType(in.DeviceType), Vendor: in.Vendor,
		StreamURL: in.StreamURL, PlaybackURL: in.PlaybackURL, Remark: in.Remark,
	}
	if in.Status != 0 {
		item.Status = in.Status
	} else {
		item.Status = 1
	}
	if err := s.repos.Surveillance.ForTenant(s.tenantID).Create(item); err != nil {
		return nil, err
	}
	return item, nil
}

func defaultDeviceType(t string) string {
	if t == "" {
		return "camera"
	}
	return t
}
