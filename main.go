// POC

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/GoScouter/sdk"
)

type NginxModule struct {}

func (module *NginxModule) Name() string {
    return "Nginx"
}

func (module *NginxModule) Description() string {
    return "Checks site information regarding Nginx"
}

func (module *NginxModule) Version() string {
    return "0.0.1"
}

func (module *NginxModule) Scout(target string) (sdk.Result, error) {
    return &NginxResult{}, nil
}

type NginxResult struct {}

func (r *NginxResult) Render() string {
    return "TODO"
}

func main() {
	module := &NginxModule{}

	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "expected a subcommand: describe or scout")
		os.Exit(2)
	}

	switch os.Args[1] {
	case "describe":
		enc := json.NewEncoder(os.Stdout)
		if err := enc.Encode(sdk.Descriptor{
			Protocol:    sdk.ProtocolVersion,
			Name:        module.Name(),
			Description: module.Description(),
			Version:     module.Version(),
		}); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

	case "scout":
		fs := flag.NewFlagSet("scout", flag.ExitOnError)
		target := fs.String("target", "", "target to scout")
		_ = fs.Parse(os.Args[2:])

		result, err := module.Scout(*target)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Print(result.Render())

	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand %q\n", os.Args[1])
		os.Exit(2)
	}
}
