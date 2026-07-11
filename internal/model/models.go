package model

import "time"

// Store 物理门店档案（OSMS 门店子域）
type Store struct {
	ID            uint64    `gorm:"primaryKey" json:"id"`
	TenantID      uint64    `gorm:"index;not null" json:"tenantId"`
	Code          string    `gorm:"size:64;not null" json:"code"`
	Name          string    `gorm:"size:128;not null" json:"name"`
	ShortName     string    `gorm:"size:64" json:"shortName"`
	Status        int8      `gorm:"default:1;not null" json:"status"`
	Phone         string    `gorm:"size:32" json:"phone"`
	Province      string    `gorm:"size:32" json:"province"`
	City          string    `gorm:"size:32" json:"city"`
	District      string    `gorm:"size:32" json:"district"`
	Address       string    `gorm:"size:255" json:"address"`
	BusinessHours string    `gorm:"size:128" json:"businessHours"`
	CoverPic      string    `gorm:"size:512" json:"coverPic"`
	Photos        []string  `gorm:"type:text;serializer:json" json:"photos"`
	GuideText     string    `gorm:"type:text" json:"guideText"`
	GuidePics     []string  `gorm:"type:text;serializer:json" json:"guidePics"`
	Longitude     float64   `gorm:"type:decimal(10,7);not null;default:0" json:"longitude"`
	Latitude      float64   `gorm:"type:decimal(10,7);not null;default:0" json:"latitude"`
	MapLabel      string    `gorm:"size:128" json:"mapLabel"`
	Remark        string    `gorm:"type:text" json:"remark"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

func (Store) TableName() string { return "stores" }

// PosOrder 收银台即时零售单
type PosOrder struct {
	ID             uint64     `gorm:"primaryKey" json:"id"`
	TenantID       uint64     `gorm:"index;not null" json:"tenantId"`
	StoreID        uint64     `gorm:"index;not null" json:"storeId"`
	OrderNo        string     `gorm:"size:32;not null" json:"orderNo"`
	Status         string     `gorm:"size:32;not null;default:pending" json:"status"`
	PaymentMethod  string     `gorm:"size:32;not null;default:''" json:"paymentMethod"`
	PayStatus      string     `gorm:"size:32;not null;default:unpaid" json:"payStatus"`
	OriginalAmount float64    `gorm:"type:decimal(14,2);not null;default:0" json:"originalAmount"`
	DiscountAmount float64    `gorm:"type:decimal(14,2);not null;default:0" json:"discountAmount"`
	TotalAmount    float64    `gorm:"type:decimal(14,2);not null;default:0" json:"totalAmount"`
	PaidAmount     float64    `gorm:"type:decimal(14,2);not null;default:0" json:"paidAmount"`
	CustomerName   string     `gorm:"size:64" json:"customerName"`
	CustomerPhone  string     `gorm:"size:32" json:"customerPhone"`
	CashierUserID  uint64     `json:"cashierUserId"`
	ReceiptType    string     `gorm:"size:16;default:small" json:"receiptType"`
	ReceiptHTML    string     `gorm:"type:text" json:"receiptHtml"`
	QRCodeURL      string     `gorm:"size:512" json:"qrCodeUrl"`
	Remark         string     `gorm:"type:text" json:"remark"`
	PaidAt         *time.Time `json:"paidAt"`
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
	Items          []PosOrderItem `gorm:"foreignKey:PosOrderID" json:"items,omitempty"`
}

func (PosOrder) TableName() string { return "pos_orders" }

type PosOrderItem struct {
	ID            uint64  `gorm:"primaryKey" json:"id"`
	TenantID      uint64  `gorm:"index;not null" json:"tenantId"`
	PosOrderID    uint64  `gorm:"index;not null" json:"posOrderId"`
	ItemType      string  `gorm:"size:16;not null;default:product" json:"itemType"` // product | service
	SkuID         uint64  `gorm:"index;not null;default:0" json:"skuId"`
	ServiceItemID uint64  `gorm:"index;not null;default:0" json:"serviceItemId"`
	ProductName   string  `gorm:"size:255;not null" json:"productName"`
	SkuCode       string  `gorm:"size:64" json:"skuCode"`
	SpecLabel     string  `gorm:"size:255" json:"specLabel"`
	Pic           string  `gorm:"size:512" json:"pic"`
	Quantity      int     `gorm:"not null" json:"quantity"`
	OriginalPrice float64 `gorm:"type:decimal(12,2);not null;default:0" json:"originalPrice"` // 原价单价
	Discount      float64 `gorm:"type:decimal(6,2);not null;default:10" json:"discount"`     // 折扣（折），10=原价，8=八折
	UnitPrice     float64 `gorm:"type:decimal(12,2);not null" json:"unitPrice"`               // 实付单价（优惠价）
	TotalAmount   float64 `gorm:"type:decimal(14,2);not null" json:"totalAmount"`
}

func (PosOrderItem) TableName() string { return "pos_order_items" }

// StoreSalesOrder 门店销售订单（非即时零售：订货后提货、派送等）
type StoreSalesOrder struct {
	ID              uint64     `gorm:"primaryKey" json:"id"`
	TenantID        uint64     `gorm:"index;not null" json:"tenantId"`
	StoreID         uint64     `gorm:"index;not null" json:"storeId"`
	OrderNo         string     `gorm:"size:32;not null" json:"orderNo"`
	OrderType       string     `gorm:"size:32;not null;default:offline" json:"orderType"`
	Status          string     `gorm:"size:32;not null;default:draft" json:"status"`
	FulfillmentType string     `gorm:"size:32;not null;default:pickup" json:"fulfillmentType"`
	CustomerName    string     `gorm:"size:64" json:"customerName"`
	CustomerPhone   string     `gorm:"size:32" json:"customerPhone"`
	ShippingAddress string     `gorm:"type:text" json:"shippingAddress"`
	TotalAmount     float64    `gorm:"type:decimal(14,2);not null;default:0" json:"totalAmount"`
	PayStatus       string     `gorm:"size:32;not null;default:unpaid" json:"payStatus"`
	NeedProcurement bool       `gorm:"not null;default:false" json:"needProcurement"`
	Remark          string     `gorm:"type:text" json:"remark"`
	CreatedBy       uint64     `json:"createdBy"`
	CreatedAt       time.Time  `json:"createdAt"`
	UpdatedAt       time.Time  `json:"updatedAt"`
	Items           []StoreSalesOrderItem `gorm:"foreignKey:SalesOrderID" json:"items,omitempty"`
}

func (StoreSalesOrder) TableName() string { return "store_sales_orders" }

type StoreSalesOrderItem struct {
	ID            uint64  `gorm:"primaryKey" json:"id"`
	TenantID      uint64  `gorm:"index;not null" json:"tenantId"`
	SalesOrderID  uint64  `gorm:"index;not null" json:"salesOrderId"`
	SkuID         uint64  `gorm:"index;not null" json:"skuId"`
	ProductName   string  `gorm:"size:255;not null" json:"productName"`
	SkuCode       string  `gorm:"size:64" json:"skuCode"`
	SpecLabel     string  `gorm:"size:255" json:"specLabel"`
	Quantity      int     `gorm:"not null" json:"quantity"`
	UnitPrice     float64 `gorm:"type:decimal(12,2);not null" json:"unitPrice"`
	TotalAmount   float64 `gorm:"type:decimal(14,2);not null" json:"totalAmount"`
}

func (StoreSalesOrderItem) TableName() string { return "store_sales_order_items" }

// ServiceOrder 维修/服务工单
type ServiceOrder struct {
	ID              uint64     `gorm:"primaryKey" json:"id"`
	TenantID        uint64     `gorm:"index;not null" json:"tenantId"`
	StoreID         uint64     `gorm:"index;not null" json:"storeId"`
	OrderNo         string     `gorm:"size:32;not null" json:"orderNo"`
	ServiceType     string     `gorm:"size:32;not null" json:"serviceType"`
	Status          string     `gorm:"size:32;not null;default:pending" json:"status"`
	CustomerName    string     `gorm:"size:64" json:"customerName"`
	CustomerPhone   string     `gorm:"size:32" json:"customerPhone"`
	DeviceInfo      string     `gorm:"size:255" json:"deviceInfo"`
	FaultDesc       string     `gorm:"type:text" json:"faultDesc"`
	AppointmentAt   *time.Time `json:"appointmentAt"`
	EngineerName    string     `gorm:"size:64" json:"engineerName"`
	EstimatedAmount float64    `gorm:"type:decimal(14,2);default:0" json:"estimatedAmount"`
	Remark          string     `gorm:"type:text" json:"remark"`
	CreatedBy       uint64     `json:"createdBy"`
	CreatedAt       time.Time  `json:"createdAt"`
	UpdatedAt       time.Time  `json:"updatedAt"`
}

func (ServiceOrder) TableName() string { return "service_orders" }

// StoreInventory 门店库存（OSMS 库存子集）
type StoreInventory struct {
	ID           uint64    `gorm:"primaryKey" json:"id"`
	TenantID     uint64    `gorm:"index;not null" json:"tenantId"`
	StoreID      uint64    `gorm:"index;not null" json:"storeId"`
	SkuID        uint64    `gorm:"index;not null" json:"skuId"`
	SkuCode      string    `gorm:"size:64" json:"skuCode"`
	ProductName  string    `gorm:"size:255" json:"productName"`
	SpecLabel    string    `gorm:"size:255" json:"specLabel"`
	Quantity     int       `gorm:"not null;default:0" json:"quantity"`
	ReservedQty  int       `gorm:"not null;default:0" json:"reservedQty"`
	SafetyStock  int       `gorm:"default:0" json:"safetyStock"`
	UpdatedAt    time.Time `json:"updatedAt"`
	CreatedAt    time.Time `json:"createdAt"`
}

func (StoreInventory) TableName() string { return "store_inventories" }

// StorePurchaseOrder 门店采购单
type StorePurchaseOrder struct {
	ID           uint64     `gorm:"primaryKey" json:"id"`
	TenantID     uint64     `gorm:"index;not null" json:"tenantId"`
	StoreID      uint64     `gorm:"index;not null" json:"storeId"`
	PoNo         string     `gorm:"size:32;not null" json:"poNo"`
	PurchaseType string     `gorm:"size:32;not null;default:stock" json:"purchaseType"`
	SupplierID   uint64     `gorm:"index" json:"supplierId"`
	SupplierName string     `gorm:"size:128" json:"supplierName"`
	RefSalesID   uint64     `json:"refSalesOrderId"`
	Status       string     `gorm:"size:32;not null;default:draft" json:"status"`
	TotalAmount  float64    `gorm:"type:decimal(14,2);not null;default:0" json:"totalAmount"`
	Remark       string     `gorm:"type:text" json:"remark"`
	CreatedBy    uint64     `json:"createdBy"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
	Items        []StorePurchaseOrderItem `gorm:"foreignKey:PurchaseOrderID" json:"items,omitempty"`
}

func (StorePurchaseOrder) TableName() string { return "store_purchase_orders" }

type StorePurchaseOrderItem struct {
	ID              uint64  `gorm:"primaryKey" json:"id"`
	TenantID        uint64  `gorm:"index;not null" json:"tenantId"`
	PurchaseOrderID uint64  `gorm:"index;not null" json:"purchaseOrderId"`
	SkuID           uint64  `gorm:"index;not null" json:"skuId"`
	ProductName     string  `gorm:"size:255;not null" json:"productName"`
	SkuCode         string  `gorm:"size:64" json:"skuCode"`
	Quantity        int     `gorm:"not null" json:"quantity"`
	UnitPrice       float64 `gorm:"type:decimal(12,2);not null" json:"unitPrice"`
	TotalAmount     float64 `gorm:"type:decimal(14,2);not null" json:"totalAmount"`
}

func (StorePurchaseOrderItem) TableName() string { return "store_purchase_order_items" }

// SurveillanceDevice 门店监控设备
type SurveillanceDevice struct {
	ID           uint64    `gorm:"primaryKey" json:"id"`
	TenantID     uint64    `gorm:"index;not null" json:"tenantId"`
	StoreID      uint64    `gorm:"index;not null" json:"storeId"`
	Name         string    `gorm:"size:128;not null" json:"name"`
	Location     string    `gorm:"size:128" json:"location"`
	DeviceType   string    `gorm:"size:32;not null;default:camera" json:"deviceType"`
	Vendor       string    `gorm:"size:64" json:"vendor"`
	StreamURL    string    `gorm:"size:512" json:"streamUrl"`
	PlaybackURL  string    `gorm:"size:512" json:"playbackUrl"`
	Status       int8      `gorm:"default:1;not null" json:"status"`
	Remark       string    `gorm:"type:text" json:"remark"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

func (SurveillanceDevice) TableName() string { return "surveillance_devices" }

// ReceiptTemplate 小票模板（电子小票优先，云打印预留）
type ReceiptTemplate struct {
	ID               uint64    `gorm:"primaryKey" json:"id"`
	TenantID         uint64    `gorm:"index;not null" json:"tenantId"`
	StoreID          uint64    `gorm:"index" json:"storeId"`
	Name             string    `gorm:"size:128;not null" json:"name"`
	ReceiptType      string    `gorm:"size:16;not null;default:small" json:"receiptType"`
	HeaderTitle      string    `gorm:"size:128" json:"headerTitle"`
	HeaderSubtitle   string    `gorm:"size:255" json:"headerSubtitle"`
	HeaderExtra      string    `gorm:"type:text" json:"headerExtra"`
	FooterThanks     string    `gorm:"size:255" json:"footerThanks"`
	FooterExtra      string    `gorm:"type:text" json:"footerExtra"`
	ShowSkuPic       bool      `gorm:"not null;default:true" json:"showSkuPic"`
	ShowStorePhone   bool      `gorm:"not null;default:true" json:"showStorePhone"`
	ShowStoreAddress bool      `gorm:"not null;default:true" json:"showStoreAddress"`
	ShowBusinessHours bool     `gorm:"not null;default:true" json:"showBusinessHours"`
	ShowCoverPic     bool      `gorm:"not null;default:false" json:"showCoverPic"`
	ShowGuideText    bool      `gorm:"not null;default:false" json:"showGuideText"`
	ShowMapLabel     bool      `gorm:"not null;default:false" json:"showMapLabel"`
	Content          string    `gorm:"type:text" json:"content"`
	IsDefault        bool      `gorm:"not null;default:false" json:"isDefault"`
	Status           int8      `gorm:"default:1;not null" json:"status"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

func (ReceiptTemplate) TableName() string { return "receipt_templates" }

// ServiceCategory 门店服务分类（本地目录，类似商品分类）
type ServiceCategory struct {
	ID        uint64    `gorm:"primaryKey" json:"id"`
	TenantID  uint64    `gorm:"index;not null" json:"tenantId"`
	ParentID  uint64    `gorm:"index;not null;default:0" json:"parentId"`
	Name      string    `gorm:"size:128;not null" json:"name"`
	Sort      int       `gorm:"not null;default:0" json:"sort"`
	Status    int8      `gorm:"default:1;not null" json:"status"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	Children  []ServiceCategory `gorm:"-" json:"children,omitempty"`
	ItemCount int64             `gorm:"-" json:"itemCount,omitempty"`
}

func (ServiceCategory) TableName() string { return "service_categories" }

// ServiceItem 门店服务项目（可在收银台与商品一起结算）
type ServiceItem struct {
	ID          uint64    `gorm:"primaryKey" json:"id"`
	TenantID    uint64    `gorm:"index;not null" json:"tenantId"`
	CategoryID  uint64    `gorm:"index;not null" json:"categoryId"`
	Code        string    `gorm:"size:64" json:"code"`
	Name        string    `gorm:"size:128;not null" json:"name"`
	Description string    `gorm:"type:text" json:"description"`
	Price       float64   `gorm:"type:decimal(12,2);not null;default:0" json:"price"`
	DurationMin int       `gorm:"not null;default:0" json:"durationMin"`
	Pic         string    `gorm:"size:512" json:"pic"`
	Sort        int       `gorm:"not null;default:0" json:"sort"`
	Status      int8      `gorm:"default:1;not null" json:"status"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
	CategoryName string   `gorm:"-" json:"categoryName,omitempty"`
}

func (ServiceItem) TableName() string { return "service_items" }
