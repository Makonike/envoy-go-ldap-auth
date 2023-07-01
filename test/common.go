package test

import (
	"fmt"
	"os"
	"os/exec"
)

func startEnvoyBind(host string, port int, baseDn, attribute string) {
	startEnvoy(host, port, baseDn, attribute, "", "", "", false, false, false, "")
}

func startEnvoySearch(host string, port int, baseDn, attribute, bindDn, bindPassword, filter string) {
	startEnvoy(host, port, baseDn, attribute, bindDn, bindPassword, filter, false, false, false, "")
}

func startEnvoyTLS(host string, port int, baseDn, attribute string) {
	startEnvoy(host, port, baseDn, attribute, "", "", "", true, false, false, "")
}

func startEnvoy(host string, port int, baseDn, attribute, bindDn, bindPassword, filter string, tls, startTLS, insecureSkipVerify bool, rootCA string) {
	generateEnvoyConfig(host, port, baseDn, attribute, bindDn, bindPassword, filter, tls, startTLS, insecureSkipVerify, rootCA)
	if startTLS {
		var err error
		err = exec.Command("sed -i \"s/host: localhost/host: $(ifconfig eth0 | awk '/inet / {print $2}')/\" envoy.yaml").Run()
		if err != nil {
			panic(fmt.Sprintf("failed to sed envoy.yaml: %v", err))
		}
		err = exec.Command("awk 'FNR==NR{a=a$0\"\\\\n\";next} /rootCA: # \"\"/{sub(/rootCA: # \"\"/, \"rootCA: \\\"\"a\"\\\"\")} 1' glauth.crt envoy.yaml > envoy.yaml.tmp && mv envoy.yaml.tmp envoy.yaml").Run()
		if err != nil {
			panic(fmt.Sprintf("failed to sed envoy.yaml: %v", err))
		}
		cmd := exec.Command("cat", "envoy.yaml")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err = cmd.Start()
		if err != nil {
			panic(fmt.Sprintf("failed to start envoy: %v", err))
		}
	}
	cmd := exec.Command("envoy", "-c", "envoy.yaml")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Start()
	if err != nil {
		panic(fmt.Sprintf("failed to start envoy: %v", err))
	}
	err = cmd.Wait()
	if err != nil {
		panic(fmt.Sprintf("failed to wait envoy: %v", err))
	}
}

func generateEnvoyConfig(host string, port int, baseDn, attribute, bindDn, bindPassword, filter string, tls, startTLS, insecureSkipVerify bool, rootCA string) {
	config := fmt.Sprintf(`
static_resources:

  listeners:
    - name: listener_0
      address:
        socket_address:
          address: 0.0.0.0
          port_value: 10000
      filter_chains:
        - filters:
            - name: envoy.filters.network.http_connection_manager
              typed_config:
                "@type": type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager
                stat_prefix: ingress_http
                access_log:
                  - name: envoy.access_loggers.stdout
                    typed_config:
                      "@type": type.googleapis.com/envoy.extensions.access_loggers.stream.v3.StdoutAccessLog
                http_filters:
                  - name: envoy.filters.http.golang
                    typed_config:
                      "@type": type.googleapis.com/envoy.extensions.filters.http.golang.v3alpha.Config
                      library_id: example
                      library_path: /etc/envoy/libgolang.so
                      plugin_name: envoy-go-ldap-auth
                      plugin_config:
                        "@type": type.googleapis.com/xds.type.v3.TypedStruct
                        value:
                          # required
                          host: %s
                          port: %d
                          baseDn: %s
                          attribute: %s
                          # optional
                          # be used in search mode
                          bindDn: %s # cn=admin,dc=example,dc=com
                          bindPassword: %s # mypassword
                          # if the filter is set, the filter application will run in search mode.
                          filter: %s # (&(objectClass=inetOrgPerson)(gidNumber=500)(uid=%%s))
                          timeout: 60 # unit is second.
                          tls: true %t # false
                          startTLS: %t # false
                          insecureSkipVerify: %t # false
                          rootCA: %s # ""

                  - name: envoy.filters.http.router
                    typed_config:
                      "@type": type.googleapis.com/envoy.extensions.filters.http.router.v3.Router
                route_config:
                  name: local_route
                  virtual_hosts:
                    - name: local_service
                      domains: ["*"]
                      routes:
                        - match:
                            prefix: "/"
                          route:
                            host_rewrite_literal: mosn.io
                            cluster: service_mosn_io

  clusters:
    - name: service_mosn_io
      type: LOGICAL_DNS
      # Comment out the following line to test on v6 networks
      dns_lookup_family: V4_ONLY
      load_assignment:
        cluster_name: service_mosn_io
        endpoints:
          - lb_endpoints:
              - endpoint:
                  address:
                    socket_address:
                      address: mosn.io
                      port_value: 443
      transport_socket:
        name: envoy.transport_sockets.tls
        typed_config:
          "@type": type.googleapis.com/envoy.extensions.transport_sockets.tls.v3.UpstreamTlsContext
          sni: mosn.io
`, host, port, baseDn, attribute, bindDn, bindPassword, filter, tls, startTLS, insecureSkipVerify, rootCA)

	// Write the configuration to the specified file
	err := os.WriteFile("envoy.yaml", []byte(config), 0644)
	if err != nil {
		panic(fmt.Sprintf("failed to write Envoy configuration to file: %v", err))
	}
}
