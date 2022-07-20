package srv

import (
	"github.com/ability-sh/abi-micro/micro"
)

func init() {
	micro.Reg("uv-user", func(name string, config interface{}) (micro.Service, error) {
		return newUserService(name, config), nil
	})
}
