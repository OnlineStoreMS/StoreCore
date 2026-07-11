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
	ReceiptType     string         `json:"receiptType"`
	CustomerName    string         `json:"customerName"`
	CustomerPhone   string         `json:"customerPhone"`
	Remark          string         `json:"remark"`
	ServiceOrderID  uint64         `json:"serviceOrderId"` // 关联服务工单结算
	Items           []OrderLineDTO `json:"items" binding:"required"`
}

type StoreSalesOrderDTO struct {
	StoreID         uint64         `json:"storeId" binding:"required"`
	FulfillmentType string         `json:"fulfillmentType"`
	CustomerName    string         `json:"customerName"`
	CustomerPhone   string         `json:"customerPhone"`
	ShippingAddress string         `json:"shippingAddress"`
	NeedProcurement bool           `json:"needProcurement"`
	Remark          string         `json:"remark"`
	Items           []OrderLineDTO `json:"items" binding:"required"`
}

type ServiceOrderLineDTO struct {
	ServiceItemID uint64 `json:"serviceItemId" binding:"required"`
	Quantity      int    `json:"quantity"`
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
	Quantity    int    `json:"quantity" binding:"required"`
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
