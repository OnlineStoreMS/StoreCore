package dto

type StoreDTO struct {
	Code          string   `json:"code" binding:"required"`
	Name          string   `json:"name" binding:"required"`
	ShortName     string   `json:"shortName"`
	Status        int8     `json:"status"`
	Phone         string   `json:"phone"`
	Province      string   `json:"province"`
	City          string   `json:"city"`
	District      string   `json:"district"`
	Address       string   `json:"address"`
	BusinessHours string   `json:"businessHours"`
	BrandLogo     string   `json:"brandLogo"`
	CoverPic      string   `json:"coverPic"`
	Photos        []string `json:"photos"`
	GuideText     string   `json:"guideText"`
	GuidePics     []string `json:"guidePics"`
	Longitude     float64  `json:"longitude"`
	Latitude      float64  `json:"latitude"`
	MapLabel      string   `json:"mapLabel"`
	Remark        string   `json:"remark"`
}

type OrderLineDTO struct {
	ItemType      string  `json:"itemType"` // product | service，默认 product
	SkuID         uint64  `json:"skuId"`
	ServiceItemID uint64  `json:"serviceItemId"`
	ProductName   string  `json:"productName" binding:"required"`
	SkuCode       string  `json:"skuCode"`
	SpecLabel     string  `json:"specLabel"`
	Pic           string  `json:"pic"`
	Quantity      int     `json:"quantity" binding:"required"`
	OriginalPrice float64 `json:"originalPrice"` // 原价；0 则用 unitPrice
	Discount      float64 `json:"discount"`      // 折扣（折），10=原价，8=八折；0 则按价推算
	UnitPrice     float64 `json:"unitPrice" binding:"required"` // 实付单价
}

type PosOrderDTO struct {
	StoreID         uint64         `json:"storeId" binding:"required"`
	PaymentMethod   string         `json:"paymentMethod"`
	IsPreview       bool           `json:"isPreview"` // 预结算单：生成明细给顾客查看，不扣库存、不收款
	IsHeld          bool           `json:"isHeld"`    // 挂单：暂存购物车，可回载继续收银
	ResumeOrderID   uint64         `json:"resumeOrderId"` // 对已有预结算/挂单/待付款单继续结算或更新
	ReceiptType     string         `json:"receiptType"`
	CustomerName    string         `json:"customerName"`
	CustomerPhone   string         `json:"customerPhone"`
	Remark          string         `json:"remark"`
	ServiceOrderID  uint64         `json:"serviceOrderId"` // 关联服务工单结算
	Items           []OrderLineDTO `json:"items" binding:"required"`
}

// PosMarkPaidDTO 确认收款（可指定支付方式；预结算/挂单继续收款时用）
type PosMarkPaidDTO struct {
	PaymentMethod string `json:"paymentMethod"`
}

type SalesServiceLineDTO struct {
	ServiceItemID uint64  `json:"serviceItemId"` // 0 表示手动添加
	ServiceName   string  `json:"serviceName"`
	ServiceCode   string  `json:"serviceCode"`
	Quantity      int     `json:"quantity"`
	OriginalPrice float64 `json:"originalPrice"`
	Discount      float64 `json:"discount"`
	UnitPrice     float64 `json:"unitPrice"`
	DurationMin   int     `json:"durationMin"`
	Pic           string  `json:"pic"`
}

type StoreSalesOrderDTO struct {
	StoreID              uint64                `json:"storeId" binding:"required"`
	FulfillmentType      string                `json:"fulfillmentType"`
	IsPreview            bool                  `json:"isPreview"` // 预结算：生成明细给顾客看，不进入履约
	CustomerName         string                `json:"customerName"`
	CustomerPhone        string                `json:"customerPhone"`
	AppointmentAt        *string               `json:"appointmentAt"` // RFC3339 / 本地时间
	PickupPersonName     string                `json:"pickupPersonName"`
	PickupPersonPhone    string                `json:"pickupPersonPhone"`
	PickupCode           string                `json:"pickupCode"`
	DeliveryType         string                `json:"deliveryType"` // huolala | errand | store_delivery
	ExpectedDeliveryAt   *string               `json:"expectedDeliveryAt"`
	ReceiverName         string                `json:"receiverName"`
	ReceiverPhone        string                `json:"receiverPhone"`
	ShippingAddress      string                `json:"shippingAddress"`
	ExpressCompany       string                `json:"expressCompany"`
	ExpressNo            string                `json:"expressNo"`
	ExpressScheduledAt   *string               `json:"expressScheduledAt"`
	NeedProcurement      bool                  `json:"needProcurement"`
	Remark               string                `json:"remark"`
	Items                []OrderLineDTO        `json:"items" binding:"required"`
	ServiceItems         []SalesServiceLineDTO `json:"serviceItems"`
}

type ServiceOrderLineDTO struct {
	ItemType      string  `json:"itemType"` // service | product，默认按字段推断
	ServiceItemID uint64  `json:"serviceItemId"`
	SkuID         uint64  `json:"skuId"`
	ProductName   string  `json:"productName"`
	SkuCode       string  `json:"skuCode"`
	SpecLabel     string  `json:"specLabel"`
	Pic           string  `json:"pic"`
	Quantity      int     `json:"quantity"`
	OriginalPrice float64 `json:"originalPrice"` // 原价；空则取目录价/实付价
	Discount      float64 `json:"discount"`      // 折，10=原价
	UnitPrice     float64 `json:"unitPrice"`     // 实付单价；服务可空（取目录价）
}

type ServiceOrderMergeReceiptDTO struct {
	IDs           []uint64 `json:"ids" binding:"required"`
	IncludeReport bool     `json:"includeReport"` // 合并时附带各工单服务报告
}

// ServiceProcessMediaDTO 过程纪录媒体
type ServiceProcessMediaDTO struct {
	URL       string `json:"url" binding:"required"`
	MediaType string `json:"mediaType"` // image | video，可空则按 URL 推断
}

// ServiceProcessRecordDTO 创建/更新服务过程纪录
type ServiceProcessRecordDTO struct {
	Phase string                   `json:"phase" binding:"required"` // before | after
	Note  string                   `json:"note"`
	Media []ServiceProcessMediaDTO `json:"media"`
}

// ServiceDocBundleDTO 合并票据与报告（单工单）
type ServiceDocBundleDTO struct {
	IncludeReceipt bool `json:"includeReceipt"`
	IncludeReport  bool `json:"includeReport"`
}

// ServiceMarkPaidDTO 线下确认收款（转账截图等）
type ServiceMarkPaidDTO struct {
	PaymentMethod   string `json:"paymentMethod"`   // transfer | cash | other
	PaymentProofURL string `json:"paymentProofUrl"` // 付款截图 URL；转账时必填
	PaidAt          string `json:"paidAt"`          // 可选，截图识别或手填；RFC3339 / 2006-01-02 15:04:05
}

// SalesMarkPaidDTO 销售单确认收款（可附付款截图与付款时间）
type SalesMarkPaidDTO struct {
	PaymentMethod   string `json:"paymentMethod"`
	PaymentProofURL string `json:"paymentProofUrl"`
	PaidAt          string `json:"paidAt"`
}

// PosOrderListFilter 收银订单列表筛选
type PosOrderListFilter struct {
	Status        string // pending|completed|preview|held
	PayStatus     string // unpaid|paid
	PaymentMethod string
	Keyword       string // 单号/顾客/电话
}

// ServiceOrderListFilter 服务工单列表筛选
type ServiceOrderListFilter struct {
	Status    string // pending|in_progress|awaiting_payment|completed|cancelled
	PayStatus string // unpaid|paid
	OrderMode string // instant|appointment
	Keyword   string
}

// SalesOrderListFilter 销售订单列表筛选
type SalesOrderListFilter struct {
	Status          string
	PayStatus       string
	FulfillmentType string // pickup|install|delivery|express
	PurchaseStatus  string
	Keyword         string
}

type ServiceOrderDTO struct {
	StoreID         uint64                `json:"storeId" binding:"required"`
	OrderMode       string                `json:"orderMode"` // instant | appointment，默认 appointment
	CustomerName    string                `json:"customerName"`
	CustomerPhone   string                `json:"customerPhone"`
	DeviceInfo      string                `json:"deviceInfo"`
	FaultDesc       string                `json:"faultDesc"`
	AppointmentAt   *string               `json:"appointmentAt"` // RFC3339
	EngineerName    string                `json:"engineerName"`
	Remark          string                `json:"remark"`
	Items           []ServiceOrderLineDTO `json:"items" binding:"required"`
	// 提醒：设计为微信消息，暂不实际发送
	ReminderEnabled bool    `json:"reminderEnabled"`
	ReminderAt      *string `json:"reminderAt"` // RFC3339；空则默认预约时间前 30 分钟
}

type InventoryAdjustDTO struct {
	StoreID     uint64 `json:"storeId" binding:"required"`
	SkuID       uint64 `json:"skuId" binding:"required"`
	SkuCode     string `json:"skuCode"`
	ProductName string `json:"productName"`
	SpecLabel   string `json:"specLabel"`
	Pic         string `json:"pic"`
	Quantity    int    `json:"quantity"` // 允许 0（盘点清零）；勿用 binding required，Gin 会把 0 当未传
	SafetyStock int    `json:"safetyStock"`
}

type StorePurchaseOrderDTO struct {
	StoreID      uint64         `json:"storeId" binding:"required"`
	PurchaseType string         `json:"purchaseType"`
	SupplierID   uint64         `json:"supplierId"`
	SupplierName string         `json:"supplierName"`
	RefSalesID   uint64         `json:"refSalesOrderId"`
	Remark       string         `json:"remark"`
	Items        []OrderLineDTO `json:"items" binding:"required"`
}

type SurveillanceDeviceDTO struct {
	StoreID     uint64 `json:"storeId" binding:"required"`
	Name        string `json:"name" binding:"required"`
	Location    string `json:"location"`
	DeviceType  string `json:"deviceType"`
	Vendor      string `json:"vendor"`
	StreamURL   string `json:"streamUrl"`
	PlaybackURL string `json:"playbackUrl"`
	Status      int8   `json:"status"`
	Remark      string `json:"remark"`
}

type StatusActionDTO struct {
	Status string `json:"status" binding:"required"`
}

type ReceiptTemplateDTO struct {
	StoreID            uint64 `json:"storeId"`
	Name               string `json:"name" binding:"required"`
	ReceiptType        string `json:"receiptType"`
	HeaderTitle        string `json:"headerTitle"`
	HeaderSubtitle     string `json:"headerSubtitle"`
	HeaderExtra        string `json:"headerExtra"`
	FooterThanks       string `json:"footerThanks"`
	FooterExtra        string `json:"footerExtra"`
	ShowSkuPic         *bool  `json:"showSkuPic"`
	ShowStorePhone     *bool  `json:"showStorePhone"`
	ShowStoreAddress   *bool  `json:"showStoreAddress"`
	ShowBusinessHours  *bool  `json:"showBusinessHours"`
	ShowBrandLogo      *bool  `json:"showBrandLogo"`
	ShowCoverPic       *bool  `json:"showCoverPic"`
	ShowGuideText      *bool  `json:"showGuideText"`
	ShowMapLabel       *bool  `json:"showMapLabel"`
	IsDefault          bool   `json:"isDefault"`
	Status             int8   `json:"status"`
}

type ServiceCategoryDTO struct {
	ParentID uint64 `json:"parentId"`
	Name     string `json:"name" binding:"required"`
	Sort     int    `json:"sort"`
	Status   int8   `json:"status"`
}

type ServiceItemDTO struct {
	CategoryID  uint64  `json:"categoryId" binding:"required"`
	Code        string  `json:"code"`
	Name        string  `json:"name" binding:"required"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
	DurationMin int     `json:"durationMin"`
	Pic         string  `json:"pic"`
	Sort        int     `json:"sort"`
	Status      int8    `json:"status"`
}

type StockTransferLineDTO struct {
	SkuID       uint64 `json:"skuId" binding:"required"`
	SkuCode     string `json:"skuCode"`
	ProductName string `json:"productName" binding:"required"`
	SpecLabel   string `json:"specLabel"`
	Pic         string `json:"pic"`
	Quantity    int    `json:"quantity" binding:"required"`
}

type StockTransferOrderDTO struct {
	StoreID         uint64                 `json:"storeId" binding:"required"`
	ExpectedAt      *string                `json:"expectedAt"`
	Remark          string                 `json:"remark"`
	RefSalesID      uint64                 `json:"refSalesOrderId"`
	Items           []StockTransferLineDTO `json:"items" binding:"required"`
	ReminderEnabled bool                   `json:"reminderEnabled"`
	ReminderAt      *string                `json:"reminderAt"`
}
