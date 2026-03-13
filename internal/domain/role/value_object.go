package role

import "encoding/json"

type Code string

const (
	Admin  Code = "admin"
	Vendor Code = "vendor"
	User   Code = "user"
	Client Code = "client"
)

func CodeFrom(s string) (Code, error) {
	switch Code(s) {
	case Admin, Vendor, User, Client:
		return Code(s), nil
	default:
		return "", ErrInvalidType
	}
}

func (t *Code) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}

	v, err := CodeFrom(s)
	if err != nil {
		return err
	}

	*t = v
	return nil
}

func (t *Code) String() string {
	return string(*t)
}
