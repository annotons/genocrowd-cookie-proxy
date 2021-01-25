# genocrowd-cookie-proxy [![Docker Repository on Quay](https://quay.io/repository/annotons/genocrowd-cookie-proxy/status "Docker Repository on Quay")](https://quay.io/repository/annotons/genocrowd-cookie-proxy)

genocrowd-cookie-proxy translate a [Genocrowd](https://githb.com/annotons/genocrowd) session cookie into a
`REMOTE_USER` identity. This allows you to use Genocrowd as your primary source of
authentication data, and provide access control to other services based on
Genocrowd, in particular Apollo.

The code is a WebSocket-aware SSL-capable HTTP reverse proxy based on
[drunken-hipster](https://github.com/joinmytalk/drunken-hipster)

genocrowd-cookie-proxy is adapted from the awesome work of Helena Rasche on [gx-cookie-proxy](https://github.com/hexylena/gx-cookie-proxy)

## Deployment

### Pre-requisites

1. You are already running some sort of proxy, such as Apache2 or NGINX
2. You have deployed a Genocrowd server

## Deployment

Download the binary from our [releases page](https://github.com/annotons/genocrowd-cookie-proxy/releases) and run it:

```console
./genocrowd-cookie-proxy \
	--genocrowdSecret 'I_LOVE_ICE_CREAM' \ # As it appears in genocrowd.ini
	--listenAddr localhost:5000 \ # Address to listen on
	--connect localhost:8080 # The backend you're connecting to
```

This will cause the proxy to:

- create a tunnel between frontend and backend
- connect to the database in order to decrypt cookies into usernames

On the first request, the proxy will check the cookie and attempt to decrypt it
based on the secret.

On subsequent requests, the proxy will check its cache for that cookie value,
improving performance. Cookies are cached for a maximum of one hour. This can
be made configurable if someone requests it.

## Configuration

Example apache2 configuration:

```apache2
ProxyPass  /gxc_proxy http://localhost:5000/gxc_proxy
<Location "/gxc_proxy">
	ProxyPassReverse http://localhost:5000/gxc_proxy
</Location>
```

This will connect to your backend service (running on `localhost:8080`), and
proxy requests to the backend. The backend service should either listen on
`/gxc_proxy/.*`, or should use completely relative paths rather than
absolute.

Note that my proxy shares a leading path component with my genocrowd
server. This is required in order to access the genocrowd session cookie
due to cookie restrictions.

The genocrowd-cookie-proxy is also configurable via environment variables:

Parameter            | Env Var               | Usage
-------------------- | -------------------   | -----------
`--genocrowdSecret`     | `GENOCROWD_SECRET`       | Genocrowd cookie secret
`--listenAddr`       | `GXC_LISTEN_ADDR`     | Proxy listening address
`--connect`          | `GXC_BACKEND_URL`     | Backend host + port to connect to
`--logLevel`         | `GXC_LOGLEVEL`        | Logging level (DBEUG, INFO (default), WARN, ERROR)
`--header`           | `GXC_HEADER`          | Header to send to backend service
`--statsd_address`   | `GXC_STATSD`          | StatsD server
`--statsd_prefix`    | `GXC_STATSD_PREFIX`   | StatsD prefix (`gxc.` by default)

# Changelog

- 0.1.0
  - First version based on gx-cookie-proxy v0.9.11

# License

MIT

# Authors

- Original Drunken Hipster Proxy - Andreas Krennmair <ak@synflood.at>
- Galaxy Portions - Helena Rasche <hxr@hx42.org>
- Genocrowd adaptation - Anthony Bretaudeau <anthony.bretaudeau@inrae.fr>
