package service

import (
	"errors"
	"time"

	"storecore/internal/dto"
	"storecore/internal/model"

	"gorm.io/gorm"
)

func (s *ServiceOrderService) Get(id uint64) (*model.ServiceOrder, error) {
	item, err := s.repos.Service.ForTenant(s.tenantID).GetByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return item, err
}

func (s *ServiceOrderService) UpdateStatus(id uint64, status string) (*model.ServiceOrder, error) {
	r := s.repos.Service.ForTenant(s.tenantID)
	item, err := r.GetByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	allowed := map[string][]string{
		"in_progress": {"pending"},
		"completed":   {"pending", "in_progress"},
		"cancelled":   {"pending", "in_progress"},
	}
	from, ok := allowed[status]
	if !ok {
		return nil, ErrBadRequest
	}
	valid := false
	for _, st := range from {
		if item.Status == st {
			valid = true
			break
		}
	}
	if !valid {
		return nil, ErrInvalidStatus
	}
	item.Status = status
	if err := r.Update(item); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *ServiceOrderService) Update(id uint64, in *dto.ServiceOrderDTO) (*model.ServiceOrder, error) {
	r := s.repos.Service.ForTenant(s.tenantID)
	item, err := r.GetByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if item.Status != "pending" {
		return nil, ErrInvalidStatus
	}
	item.ServiceType = in.ServiceType
	item.CustomerName = in.CustomerName
	item.CustomerPhone = in.CustomerPhone
	item.DeviceInfo = in.DeviceInfo
	item.FaultDesc = in.FaultDesc
	item.EngineerName = in.EngineerName
	item.EstimatedAmount = in.EstimatedAmount
	item.Remark = in.Remark
	item.AppointmentAt = nil
	if in.AppointmentAt != nil && *in.AppointmentAt != "" {
		t, err := time.Parse(time.RFC3339, *in.AppointmentAt)
		if err == nil {
			item.AppointmentAt = &t
		}
	}
	if err := r.Update(item); err != nil {
		return nil, err
	}
	return item, nil
}
