envoy-go-ldap-auth
==================

This is a simple LDAP auth filter for envoy written in go. Only requests that pass the LDAP server's authentication will be proxied to the upstream service.

During this process, we can optimize the system by implementing user information caching with a duration defined by `config.cache_ttl`. This approach will help reduce the frequency of LDAP server access. If it is set to 0, caching is disabled by default.

In terms of caching, we utilize [bigcache](https://github.com/allegro/bigcache), which demonstrates exceptional performance in the evicting cache domain.

## Status

This is under active development and is not ready for production use.

## Usage

The client set credentials in `Authorization` header in the following format:

```Plaintext
credentials := Basic base64(username:password)
```

An example of the `Authorization` header is as follows (`aGFja2Vyczpkb2dvb2Q=`, which is the base64-encoded value of `hackers:dogood`):

```Plaintext
Authorization: Basic aGFja2Vyczpkb2dvb2Q=
```

Configure your envoy.yaml, include required fields: host, port, base_dn and attribute.

```yaml
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
          host: localhost
          port: 389
          base_dn: dc=example,dc=com
          attribute: cn
          # optional
          # be used in search mode
          bind_dn: 
          bind_password: 
          # if the filter is set, the filter application will run in search mode.
          filter: 
          cache_ttl: 0
          timeout: 60
```

Then, you can start your filter.

```bash
make build
```

```bash
make run 
```

## Test

This test case is based on glauth and can be utilized to evaluate your filter.

Firstly, download [glauth](https://github.com/glauth/glauth/releases), and change its [sample config file](https://github.com/glauth/glauth/blob/master/v2/sample-simple.cfg).

sample.yaml

```yaml
[ldap]
  enabled = true
  # run on a non privileged port
  listen = "192.168.64.1:3893" # 192.168.64.1 is your local network IP. Please synchronize it with envoy.yaml.
```

envoy.yaml
```yaml
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
          host: 192.168.64.1
          port: 3893
          base_dn: dc=glauth,dc=com
          attribute: cn
          # optional
          # be used in search mode
          bind_dn: cn=serviceuser,ou=svcaccts,dc=glauth,dc=com
          bind_password: mysecret
          # if the filter is set, the filter application will run in search mode.
          filter: (cn=%s)
          cache_ttl: 0
          timeout: 60
```

Then, start glauth.

```bash
./glauth -c sample.yaml
```

Once you have activated the filter, you can execute the following command to test it.

```bash
make test
```

The following are the output results.

```bash
$ make test
curl -s -I 'http://localhost:10000/'
HTTP/1.1 401 Unauthorized
content-length: 16
content-type: text/plain
date: Sun, 28 May 2023 16:56:18 GMT
server: envoy

curl -s -I 'http://localhost:10000/' -H 'Authorization: Basic dW5rbm93bjpkb2dvb2Q=' # generated by `echo -n "unknown:dogood" | base64`
HTTP/1.1 401 Unauthorized
content-length: 28
content-type: text/plain
date: Sun, 28 May 2023 16:56:18 GMT
server: envoy

curl -s -I 'http://localhost:10000/' -H 'Authorization: Basic aGFja2Vyczp1bmtub3du' # generated by `echo -n "hackers:unknown" | base64`
HTTP/1.1 401 Unauthorized
content-length: 28
content-type: text/plain
date: Sun, 28 May 2023 16:56:18 GMT
server: envoy

curl -s -I 'http://localhost:10000/' -H 'Authorization: Basic aGFja2Vyczpkb2dvb2Q=' # generated by `echo -n "hackers:dogood" | base64`
HTTP/1.1 200 OK
date: Sun, 28 May 2023 16:56:20 GMT
content-type: text/html; charset=utf-8
content-length: 12725
permissions-policy: interest-cohort=()
last-modified: Sat, 06 May 2023 08:09:59 GMT
access-control-allow-origin: *
strict-transport-security: max-age=31556952
etag: "64560b57-31b5"
expires: Sun, 28 May 2023 17:04:05 GMT
cache-control: max-age=600
x-proxy-cache: MISS
x-github-request-id: FD8A:64E8:E68E8:F6229:6473872C
accept-ranges: bytes
via: 1.1 varnish
age: 136
x-served-by: cache-hkg17920-HKG
x-cache: HIT
x-cache-hits: 1
x-timer: S1685292981.889966,VS0,VE30
vary: Accept-Encoding
x-fastly-request-id: 5c0885547b1360efd368516eb77213e7a43586ba
server: envoy
x-envoy-upstream-service-time: 1974
```


## Bind Mode and Search Mode

If no filter is specified in its configuration, the middleware runs in the default bind mode, meaning it tries to make a simple bind request to the LDAP server with the credentials provided in the request headers. If the bind succeeds, the middleware forwards the request, otherwise it returns a `401 Unauthorized` status code.

If a filter query is specified in the middleware configuration, and the Authentication Source referenced has a `bindDN` and a `bindPassword`, then the middleware runs in search mode. In this mode, a search query with the given filter is issued to the LDAP server before trying to bind. If result of this search returns only 1 record, it tries to issue a bind request with this record, otherwise it aborts a `401 Unauthorized` status code.

## Config

### Required

- host, string, default "localhost", required

Host on which the LDAP server is running.

- port, number, default 389, required

TCP port where the LDAP server is listening. 389 is the default port for LDAP.

- base_dn, string, "dc=example,dc=com", required

The `baseDN` option should be set to the base domain name that should be used for bind and search queries.

- attribute, string, default "cn", required

Attribute to be used to search the user; e.g., “cn”.

### Optional

- filter, string, default ""

If not empty, the middleware will run in search mode, filtering search results with the given query.

Filter queries can use the `%s` placeholder that is replaced by the username provided in the `Authorization` header of the request. For example: `(&(objectClass=inetOrgPerson)(gidNumber=500)(uid=%s))`, `(cn=%s)`.

- bind_dn, string, default ""

The domain name to bind to in order to authenticate to the LDAP server when running on search mode. Leaving this empty with search mode means binds are anonymous, which is rarely expected behavior. It is not used when running in bind_mode.

- bind_password, string, default ""

The password corresponding to the `bindDN` specified when running in search mode, used in order to authenticate to the LDAP server.

- cache_ttl, number, default 0

Cache expiry time in seconds. If it is set to 0, caching is disabled by default.

- timeout, number, default 60

An optional timeout in seconds when waiting for connection with LDAP server.

