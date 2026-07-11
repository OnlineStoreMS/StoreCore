package service

import (
	"errors"
	"strings"

	"storecore/internal/dto"
	"storecore/internal/model"
	"storecore/internal/repo"

	"gorm.io/gorm"
)

type ServiceCatalogService struct {
	repos    *repo.Repos
	tenantID uint64
}

func NewServiceCatalogService(repos *repo.Repos) *ServiceCatalogService {
	return &ServiceCatalogService{repos: repos}
}

func (s *ServiceCatalogService) ForTenant(tenantID uint64) *ServiceCatalogService {
	return &ServiceCatalogService{repos: s.repos, tenantID: repo.NormalizeTenantID(tenantID)}
}

func (s *ServiceCatalogService) CategoryTree() ([]model.ServiceCategory, error) {
	r := s.repos.ServiceCatalog.ForTenant(s.tenantID)
	flat, err := r.ListCategories()
	if err != nil {
		return nil, err
	}
	counts, _ := r.CountItemsByCategory()
	for i := range flat {
		flat[i].ItemCount = counts[flat[i].ID]
	}
	return buildServiceCategoryTree(flat, 0), nil
}

func (s *ServiceCatalogService) ListCategories() ([]model.ServiceCategory, error) {
	return s.repos.ServiceCatalog.ForTenant(s.tenantID).ListCategories()
}

func (s *ServiceCatalogService) CreateCategory(in *dto.ServiceCategoryDTO) (*model.ServiceCategory, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, ErrBadRequest
	}
	item := &model.ServiceCategory{
		ParentID: in.ParentID,
		Name:     name,
		Sort:     in.Sort,
		Status:   in.Status,
	}
	if item.Status == 0 {
		item.Status = 1
	}
	if err := s.repos.ServiceCatalog.ForTenant(s.tenantID).CreateCategory(item); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *ServiceCatalogService) UpdateCategory(id uint64, in *dto.ServiceCategoryDTO) (*model.ServiceCategory, error) {
	r := s.repos.ServiceCatalog.ForTenant(s.tenantID)
	item, err := r.GetCategory(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, ErrBadRequest
	}
	item.ParentID = in.ParentID
	item.Name = name
	item.Sort = in.Sort
	item.Status = in.Status
	if err := r.UpdateCategory(item); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *ServiceCatalogService) DeleteCategory(id uint64) error {
	r := s.repos.ServiceCatalog.ForTenant(s.tenantID)
	if _, err := r.GetCategory(id); errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	} else if err != nil {
		return err
	}
	err := r.DeleteCategory(id)
	if errors.Is(err, gorm.ErrForeignKeyViolated) {
		return errors.New("请先删除子分类及该分类下的服务项目")
	}
	return err
}

func (s *ServiceCatalogService) ListItems(categoryID uint64, keyword string, status *int8, page, pageSize int) ([]model.ServiceItem, int64, error) {
	r := s.repos.ServiceCatalog.ForTenant(s.tenantID)
	list, total, err := r.ListItems(categoryID, keyword, status, page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	cats, _ := r.ListCategories()
	nameByID := make(map[uint64]string, len(cats))
	for _, c := range cats {
		nameByID[c.ID] = c.Name
	}
	for i := range list {
		list[i].CategoryName = nameByID[list[i].CategoryID]
	}
	return list, total, nil
}

func (s *ServiceCatalogService) GetItem(id uint64) (*model.ServiceItem, error) {
	item, err := s.repos.ServiceCatalog.ForTenant(s.tenantID).GetItem(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return item, err
}

func (s *ServiceCatalogService) CreateItem(in *dto.ServiceItemDTO) (*model.ServiceItem, error) {
	if err := s.validateItemInput(in); err != nil {
		return nil, err
	}
	item := &model.ServiceItem{
		CategoryID:  in.CategoryID,
		Code:        strings.TrimSpace(in.Code),
		Name:        strings.TrimSpace(in.Name),
		Description: strings.TrimSpace(in.Description),
		Price:       in.Price,
		DurationMin: in.DurationMin,
		Pic:         strings.TrimSpace(in.Pic),
		Sort:        in.Sort,
		Status:      in.Status,
	}
	if item.Status == 0 {
		item.Status = 1
	}
	if err := s.repos.ServiceCatalog.ForTenant(s.tenantID).CreateItem(item); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *ServiceCatalogService) UpdateItem(id uint64, in *dto.ServiceItemDTO) (*model.ServiceItem, error) {
	if err := s.validateItemInput(in); err != nil {
		return nil, err
	}
	r := s.repos.ServiceCatalog.ForTenant(s.tenantID)
	item, err := r.GetItem(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	item.CategoryID = in.CategoryID
	item.Code = strings.TrimSpace(in.Code)
	item.Name = strings.TrimSpace(in.Name)
	item.Description = strings.TrimSpace(in.Description)
	item.Price = in.Price
	item.DurationMin = in.DurationMin
	item.Pic = strings.TrimSpace(in.Pic)
	item.Sort = in.Sort
	item.Status = in.Status
	if err := r.UpdateItem(item); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *ServiceCatalogService) DeleteItem(id uint64) error {
	r := s.repos.ServiceCatalog.ForTenant(s.tenantID)
	if _, err := r.GetItem(id); errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	} else if err != nil {
		return err
	}
	return r.DeleteItem(id)
}

func (s *ServiceCatalogService) validateItemInput(in *dto.ServiceItemDTO) error {
	if in.CategoryID == 0 || strings.TrimSpace(in.Name) == "" {
		return ErrBadRequest
	}
	if _, err := s.repos.ServiceCatalog.ForTenant(s.tenantID).GetCategory(in.CategoryID); errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrBadRequest
	} else if err != nil {
		return err
	}
	return nil
}

func buildServiceCategoryTree(items []model.ServiceCategory, parentID uint64) []model.ServiceCategory {
	out := make([]model.ServiceCategory, 0)
	for _, item := range items {
		if item.ParentID != parentID {
			continue
		}
		item.Children = buildServiceCategoryTree(items, item.ID)
		out = append(out, item)
	}
	return out
}
