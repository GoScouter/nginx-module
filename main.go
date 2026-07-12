// POC

package module

import "github.com/GoScouter/sdk"

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

