## fluid

Inspect and diagnose Fluid-managed datasets

### Synopsis

fluid is a CLI for the Fluid project.

It provides commands to inspect and diagnose Fluid-managed Datasets and their
associated Kubernetes resources (Runtimes, Pods, PVCs, PVs, Services, etc.).

Install: copy the binary to a directory in your PATH as 'fluid'.
Usage:   fluid <subcommand>

### Examples

```
  # List resources owned by a Dataset
  fluid inspect my-dataset -n default

  # Collect diagnostic data and archive it
  fluid diagnose my-dataset -n default --archive

  # Print version
  fluid version
```

### Options

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
  -h, --help                           help for fluid
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
* [fluid inspect](fluid_inspect.md)	 - List all Kubernetes resources associated with a Fluid Dataset
* [fluid version](fluid_version.md)	 - Print the version of fluid

