package main

import (
	"net"
	"os"

	"github.com/dogmatiq/imbue"
)

type environment imbue.Name[[]string]

func init() {
	imbue.With1Named[environment](
		container,
		func(
			ctx imbue.Context,
			lis imbue.ByName[askpassListener, net.Listener],
		) ([]string, error) {
			bin, err := os.Executable()
			if err != nil {
				return nil, err
			}

			env := append(
				os.Environ(),
				"GIT_CONFIG_SYSTEM=",
				"GIT_CONFIG_GLOBAL=",
				"GIT_ASKPASS="+bin,
				"ASKPASS_ADDR="+lis.Value().Addr().String(),
			)

			return env, nil
		},
	)

}
