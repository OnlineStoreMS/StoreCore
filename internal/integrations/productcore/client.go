package productcore

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

type SkuSearchItem struct {
	ProductID     uint64            `json:"productId"`
	ProductName   string            `json:"productName"`
	MaterialCode  string            `json:"materialCode"`
	ProductSn     string            `json:"productSn"`
	ProductPic    string            `json:"productPic"`
	BrandName     string            `json:"brandName"`
	CategoryName  string            `json:"categoryName"`
	PublishStatus int8              `json:"publishStatus"`
	SkuID         uint64            `json:"skuId"`
	SkuCode       string            `json:"skuCode"`
	Specs         map[string]string `json:"specs"`
	SpecLabel     string            `json:"specLabel"`
	Price         float64           `json:"price"`
	Stock         int               `json:"stock"`
	Pic           string            `json:"pic"`
}

type CategoryItem struct {
	ID           uint64         `json:"id,omitempty"`
	ParentID     uint64         `json:"parentId"`
	Name         string         `json:"name"`
	Level        int            `json:"level"`
	Sort         int            `json:"sort"`
	ShowStatus   int8           `json:"showStatus"`
	ProductCount int64          `json:"productCount,omitempty"`
	Children     []CategoryItem `json:"children,omitempty"`
}

type ProductListItem struct {
	ID            uint64  `json:"id,omitempty"`
	Name          string  `json:"name"`
	Pic           string  `json:"pic"`
	Price         float64 `json:"price"`
	Stock         int     `json:"stock"`
	SkuCount      int     `json:"skuCount,omitempty"`
	BrandID       uint64  `json:"brandId"`
	CategoryID    uint64  `json:"categoryId"`
	CategoryName  string  `json:"categoryName,omitempty"`
	MaterialCode  string  `json:"materialCode"`
	PublishStatus int8    `json:"publishStatus"`
}

type BrandItem struct {
	ID   uint64 `json:"id"`
	Name string `json:"name"`
}

type GroupItem struct {
	ID   uint64 `json:"id"`
	Name string `json:"name"`
}

type SkuItem struct {
	ID      uint64            `json:"id,omitempty"`
	SkuCode string            `json:"skuCode"`
	Specs   map[string]string `json:"specs"`
	Price   float64           `json:"price"`
	Stock   int               `json:"stock"`
	Pic     string            `json:"pic,omitempty"`
}

type ProductSkus struct {
	ID       uint64    `json:"id"`
	Name     string    `json:"name"`
	Pic      string    `json:"pic,omitempty"`
	SkuCount int       `json:"skuCount"`
	Price    float64   `json:"price"`
	Skus     []SkuItem `json:"skus"`
}

type pagePayload[T any] struct {
	List     []T   `json:"list"`
	Total    int64 `json:"total"`
	Page     int   `json:"page"`
	PageSize int   `json:"pageSize"`
}

type apiBody struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func (c *Client) SearchSkus(ctx context.Context, bearerToken, keyword string, page, pageSize int) ([]SkuSearchItem, int64, error) {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return nil, 0, fmt.Errorf("keyword required")
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	q := url.Values{}
	q.Set("keyword", keyword)
	q.Set("page", strconv.Itoa(page))
	q.Set("pageSize", strconv.Itoa(pageSize))
	var pageData pagePayload[SkuSearchItem]
	if err := c.get(ctx, bearerToken, "/api/v1/admin/super-search?"+q.Encode(), &pageData); err != nil {
		return nil, 0, err
	}
	if pageData.List == nil {
		pageData.List = []SkuSearchItem{}
	}
	return pageData.List, pageData.Total, nil
}

func (c *Client) GetCategoryTree(ctx context.Context, bearerToken string) ([]CategoryItem, error) {
	var tree []CategoryItem
	if err := c.get(ctx, bearerToken, "/api/v1/admin/categories/tree", &tree); err != nil {
		return nil, err
	}
	if tree == nil {
		tree = []CategoryItem{}
	}
	return tree, nil
}

func (c *Client) ListProducts(ctx context.Context, bearerToken, keyword string, categoryID, brandID, groupID uint64, page, pageSize int) ([]ProductListItem, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 24
	}
	q := url.Values{}
	q.Set("page", strconv.Itoa(page))
	q.Set("pageSize", strconv.Itoa(pageSize))
	q.Set("publishStatus", "1")
	if keyword = strings.TrimSpace(keyword); keyword != "" {
		q.Set("keyword", keyword)
	}
	if categoryID > 0 {
		q.Set("categoryId", strconv.FormatUint(categoryID, 10))
	}
	if brandID > 0 {
		q.Set("brandId", strconv.FormatUint(brandID, 10))
	}
	if groupID > 0 {
		q.Set("groupId", strconv.FormatUint(groupID, 10))
	}
	var pageData pagePayload[ProductListItem]
	if err := c.get(ctx, bearerToken, "/api/v1/admin/products?"+q.Encode(), &pageData); err != nil {
		return nil, 0, err
	}
	if pageData.List == nil {
		pageData.List = []ProductListItem{}
	}
	return pageData.List, pageData.Total, nil
}

func (c *Client) ListBrands(ctx context.Context, bearerToken string) ([]BrandItem, error) {
	var list []BrandItem
	if err := c.get(ctx, bearerToken, "/api/v1/admin/brands", &list); err != nil {
		return nil, err
	}
	if list == nil {
		list = []BrandItem{}
	}
	return list, nil
}

func (c *Client) ListGroups(ctx context.Context, bearerToken string) ([]GroupItem, error) {
	var list []GroupItem
	if err := c.get(ctx, bearerToken, "/api/v1/admin/groups", &list); err != nil {
		return nil, err
	}
	if list == nil {
		list = []GroupItem{}
	}
	return list, nil
}

// CollectSkuIDs 按品牌/分类/分组拉取上架商品对应的全部 SKU ID（用于门店库存筛选）。
func (c *Client) CollectSkuIDs(ctx context.Context, bearerToken string, brandID, categoryID, groupID uint64) ([]uint64, error) {
	if brandID == 0 && categoryID == 0 && groupID == 0 {
		return nil, nil
	}
	seen := map[uint64]struct{}{}
	var skuIDs []uint64
	page := 1
	const pageSize = 100
	for {
		list, total, err := c.ListProducts(ctx, bearerToken, "", categoryID, brandID, groupID, page, pageSize)
		if err != nil {
			return nil, err
		}
		for _, p := range list {
			detail, err := c.GetProductSkus(ctx, bearerToken, p.ID)
			if err != nil {
				return nil, err
			}
			for _, sku := range detail.Skus {
				if _, ok := seen[sku.ID]; ok {
					continue
				}
				seen[sku.ID] = struct{}{}
				skuIDs = append(skuIDs, sku.ID)
			}
		}
		if int64(page*pageSize) >= total || len(list) == 0 {
			break
		}
		page++
		if page > 50 {
			break
		}
	}
	return skuIDs, nil
}

func (c *Client) GetProductSkus(ctx context.Context, bearerToken string, productID uint64) (*ProductSkus, error) {
	if productID == 0 {
		return nil, fmt.Errorf("product id required")
	}
	var item ProductSkus
	path := fmt.Sprintf("/api/v1/admin/products/%d/skus", productID)
	if err := c.get(ctx, bearerToken, path, &item); err != nil {
		return nil, err
	}
	if item.Skus == nil {
		item.Skus = []SkuItem{}
	}
	return &item, nil
}

func (c *Client) get(ctx context.Context, bearerToken, path string, dest any) error {
	reqURL := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return err
	}
	if bearerToken != "" {
		req.Header.Set("Authorization", bearerToken)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("productcore request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("productcore http %d: %s", resp.StatusCode, truncate(string(body), 200))
	}

	var wrapped apiBody
	if err := json.Unmarshal(body, &wrapped); err != nil {
		return fmt.Errorf("productcore decode: %w", err)
	}
	if wrapped.Code != 200 {
		msg := wrapped.Message
		if msg == "" {
			msg = "productcore error"
		}
		return fmt.Errorf("%s", msg)
	}
	if err := json.Unmarshal(wrapped.Data, dest); err != nil {
		return fmt.Errorf("productcore data decode: %w", err)
	}
	return nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
