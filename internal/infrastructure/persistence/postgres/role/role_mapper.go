package role

import "github.com/HiroLiang/goat-server/internal/domain/role"

func ToDomain(record *RoleRecord) (*role.Role, error) {
	return &role.Role{
		ID:        record.ID,
		Code:      record.Code,
		Creator:   record.Creator,
		CreateAt:  record.CreatedAt,
		UpdatedAt: record.UpdatedAt,
	}, nil
}

func ToRecord(role *role.Role) *RoleRecord {
	return &RoleRecord{
		ID:        role.ID,
		Code:      role.Code,
		Creator:   role.Creator,
		CreatedAt: role.CreateAt,
		UpdatedAt: role.UpdatedAt,
	}
}
