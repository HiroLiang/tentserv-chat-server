package device

import "github.com/HiroLiang/goat-server/internal/domain/shared"

// ID is an alias for shared.DeviceID, used by other domain packages.
type ID = shared.DeviceID

// Platform represents the operating system of a device.
type Platform string

const (
	Android Platform = "android"
	IOS     Platform = "ios"
	Windows Platform = "windows"
	MacOS   Platform = "macos"
	Linux   Platform = "linux"
	Unknown Platform = "unknown"
)

// ParsePlatform validates and returns a Platform value.
func ParsePlatform(s string) (Platform, error) {
	switch Platform(s) {
	case Android, IOS, Windows, MacOS, Linux, Unknown:
		return Platform(s), nil
	}
	return "", ErrInvalidPlatform
}
func (p *Platform) String() string {
	return string(*p)
}
