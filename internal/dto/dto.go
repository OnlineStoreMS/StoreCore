package dto

type StoreDTO struct {
	Code          string `json:"code" binding:"required"`
	Name          string `json:"name" binding:"required"`
	ShortName     string `json:"shortName"`
	Status        int8   `json:"status"`
	Phone         string `json:"phone"`
	Province      string `json:"province"`
	City          string `json:"city"`
	District      string `json:"district"`
	Address       string `json:"address"`
	BusinessHours string `json:"businessHours"`
	Remark        string `json:"remark"`
}

type OrderLineDTO struct {
	SkuID       uint64  `json:"skuId" binding:"required"`
	ProductName string  `json:"productName" binding:"required"`
	SkuCode     string  `json:"skuCode"`
	SpecLabel   string  `json:"specLabel"`
	Quantity    int     `json:"quantity" binding:"required"`
	UnitPrice   float64 `json:"unitPrice" binding:"required"`
}

type PosOrderDTO struct {
	StoreID       uint64         `json:"storeId" binding:"required"`
	PaymentMethod string         `json:"paymentMethod" binding:"required"`
	ReceiptType   string         `json:"receiptType"`
	CustomerName  string         `json:"customerName"`
	CustomerPhone string         `json:"customerPhone"`
	Remark        string         `json:"remark"`
	Items         []OrderLineDTO `json:"items" binding:"required"`
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

type ServiceOrderDTO struct {
	StoreID         uint64  `json:"storeId" binding:"required"`
	ServiceType     string  `json:"serviceType" binding:"required"`
	CustomerName    string  `json:"customerName"`
	CustomerPhone   string  `json:"customerPhone"`
	DeviceInfo      string  `json:"deviceInfo"`
	FaultDesc       string  `json:"faultDesc"`
	AppointmentAt   *string `json:"appointmentAt"`
	EngineerName    string  `json:"engineerName"`
	EstimatedAmount float64 `json:"estimatedAmount"`
	Remark          string  `json:"remark"`
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
