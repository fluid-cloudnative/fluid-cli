## fluid diagnose config

Manage diagnose AI/LLM settings

### Synopsis

Manage Fluid diagnose AI settings stored in ~/.fluid/config.

Settings are used for OpenAI-compatible LLM analysis during fluid diagnose.
Prefer FLUID_LLM_API_KEY for secrets instead of storing apiKey in the config file.

### Options

```
  -h, --help   help for config
```

### Options inherited from parent commands

```
      --as string                      Username to impersonate for the operation. User could be a regular user or a service account in a namespace.
      --as-group stringArray           Group to impersonate for the operation, this flag can be repeated to specify multiple groups.
      --as-uid string                  UID to impersonate for the operation.
      --cache-dir string               Default cache directory (default "~/.kube/cache")
      --certificate-authority string   Path to a cert file for the certificate authority
      --client-certificate string      Path to a client certificate file for TLS
      --client-key string              Path to a client key file for TLS
      --cluster string                 The name of the kubeconfig cluster to use
      --context string                 The name of the kubeconfig context to use
      --disable-compression            If true, opt-out of response compression for all requests to the server
      --insecure-skip-tls-verify       If true, the server's certificate will not be checked for validity. This will make your HTTPS connections insecure
      --kubeconfig string              Path to the kubeconfig file to use for CLI requests.
  -n, --namespace string               If present, the namespace scope for this CLI request
      --request-timeout string         The length of time to wait before giving up on a single server request. Non-zero values should contain a corresponding time unit (e.g. 1s, 2m, 3h). A value of zero means don't timeout requests. (default "0")
  -s, --server string                  The address and port of the Kubernetes API server
      --tls-server-name string         Server name to use for server certificate validation. If it is not provided, the hostname used to contact the server is used
      --token string                   Bearer token for authentication to the API server
      --user string                    The name of the kubeconfig user to use
```

### SEE ALSO

* [fluid diagnose](fluid_diagnose.md)	 - Collect diagnostic data for a Fluid Dataset and its Runtime(s)
* [fluid diagnose config get](fluid_diagnose_config_get.md)	 - Print a diagnose LLM configuration value
* [fluid diagnose config set](fluid_diagnose_config_set.md)	 - Set a diagnose LLM configuration value
* [fluid diagnose config unset](fluid_diagnose_config_unset.md)	 - Remove a diagnose LLM configuration value
* [fluid diagnose config view](fluid_diagnose_config_view.md)	 - Show the diagnose section of ~/.fluid/config

