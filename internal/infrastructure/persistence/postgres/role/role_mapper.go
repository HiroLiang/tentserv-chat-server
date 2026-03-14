package role

import "github.com/HiroLiang/goat-server/internal/domain/role"

func ToDomain(record *RoleRecord) (*role.Role, error) {
	r := &role.Role{
		ID:        record.ID,
		Code:      record.Code,
		Name:      record.Name,
		CreateAt:  record.CreatedAt,
		UpdatedAt: record.UpdatedAt,
	}
	if record.Description != nil {
		r.Description = *record.Description
	}
	if record.CreatedBy != nil {
		r.Creator = *record.CreatedBy
	}
	return r, nil
}

func ToRecord(r *role.Role) *RoleRecord {
	rec := &RoleRecord{
		ID:        r.ID,
		Code:      r.Code,
		Name:      r.Name,
		CreatedAt: r.CreateAt,
		UpdatedAt: r.UpdatedAt,
	}
	if r.Description != "" {
		rec.Description = &r.Description
	}
	if r.Creator != 0 {
		rec.CreatedBy = &r.Creator
	}
	return rec
}
