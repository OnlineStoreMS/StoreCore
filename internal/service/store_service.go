package service

import (
	"errors"
	"strings"

	"storecore/internal/dto"
	"storecore/internal/model"
	"storecore/internal/repo"

	"gorm.io/gorm"
)

type StoreService struct {
	repos    *repo.Repos
	tenantID uint64
}

func NewStoreService(repos *repo.Repos) *StoreService {
	return &StoreService{repos: repos}
}

func (s *StoreService) ForTenant(tenantID uint64) *StoreService {
	return &StoreService{repos: s.repos, tenantID: repo.NormalizeTenantID(tenantID)}
}

func (s *StoreService) List(keyword string, page, pageSize int) ([]model.Store, int64, error) {
	return s.repos.Store.ForTenant(s.tenantID).List(keyword, page, pageSize)
}

func (s *StoreService) Get(id uint64) (*model.Store, error) {
	item, err := s.repos.Store.ForTenant(s.tenantID).GetByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return item, err
}

func (s *StoreService) Create(in *dto.StoreDTO) (*model.Store, error) {
	r := s.repos.Store.ForTenant(s.tenantID)
	if _, err := r.GetByCode(in.Code); err == nil {
		return nil, ErrDuplicateCode
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	item := dtoToStore(in)
	if err := r.Create(item); err != nil {
		if isDuplicateKey(err) {
			return nil, ErrDuplicateCode
		}
		return nil, err
	}
	return item, nil
}

func (s *StoreService) Update(id uint64, in *dto.StoreDTO) (*model.Store, error) {
	r := s.repos.Store.ForTenant(s.tenantID)
	item, err := r.GetByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if in.Code != item.Code {
		if other, err := r.GetByCode(in.Code); err == nil && other.ID != id {
			return nil, ErrDuplicateCode
		} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	}
	applyStoreDTO(item, in)
	if err := r.Update(item); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *StoreService) Delete(id uint64) error {
	if err := s.repos.Store.ForTenant(s.tenantID).Delete(id); err != nil {
		return err
	}
	return nil
}

func dtoToStore(in *dto.StoreDTO) *model.Store {
	item := &model.Store{
		Code: in.Code, Name: in.Name, ShortName: in.ShortName,
		Phone: in.Phone, Province: in.Province, City: in.City,
		District: in.District, Address: in.Address,
		BusinessHours: in.BusinessHours, Remark: in.Remark,
	}
	if in.Status != 0 {
		item.Status = in.Status
	} else {
		item.Status = 1
	}
	return item
}

func applyStoreDTO(item *model.Store, in *dto.StoreDTO) {
	item.Code = in.Code
	item.Name = in.Name
	item.ShortName = in.ShortName
	item.Phone = in.Phone
	item.Province = in.Province
	item.City = in.City
	item.District = in.District
	item.Address = in.Address
	item.BusinessHours = in.BusinessHours
	item.Remark = in.Remark
	if in.Status != 0 {
		item.Status = in.Status
	}
}

func isDuplicateKey(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate") || strings.Contains(msg, "unique")
}
