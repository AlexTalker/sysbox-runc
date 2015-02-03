// +build linux

package network

import (
	"errors"

	"github.com/docker/libcontainer/configs"
)

var (
	ErrNotValidStrategyType = errors.New("not a valid network strategy type")
)

var strategies = map[string]NetworkStrategy{
	"veth":     &Veth{},
	"loopback": &Loopback{},
}

// NetworkStrategy represents a specific network configuration for
// a container's networking stack
type NetworkStrategy interface {
	Create(*configs.Network, int, *configs.NetworkState) error
	Initialize(*configs.Network, *configs.NetworkState) error
}

// GetStrategy returns the specific network strategy for the
// provided type.  If no strategy is registered for the type an
// ErrNotValidStrategyType is returned.
func GetStrategy(tpe string) (NetworkStrategy, error) {
	s, exists := strategies[tpe]
	if !exists {
		return nil, ErrNotValidStrategyType
	}
	return s, nil
}
