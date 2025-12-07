# DynDNS Proxy for Cloudflare

[![CI](https://github.com/timeteus/dyndns-cloudflare-proxy/actions/workflows/ci.yml/badge.svg)](https://github.com/timeteus/dyndns-cloudflare-proxy/actions/workflows/ci.yml)
[![Release](https://github.com/timeteus/dyndns-cloudflare-proxy/actions/workflows/release.yml/badge.svg)](https://github.com/timeteus/dyndns-cloudflare-proxy/actions/workflows/release.yml)
[![Docker Pulls](https://img.shields.io/docker/pulls/timeteus/dyndns-cloudflare-proxy)](https://hub.docker.com/r/timeteus/dyndns-cloudflare-proxy)

A lightweight Go-based DynDNS server that acts as a proxy/router to update DNS records on Cloudflare using the Cloudflare REST API. This service implements the DynDNS protocol, allowing you to use standard DynDNS clients to update your Cloudflare DNS records.

## Quick Start

### Using Docker Hub (Recommended)

Pull and run the pre-built image from Docker Hub:

```bash
docker run -d \
  -p 8080:8080 \
  -e CLOUDFLARE_API_TOKEN=your_api_token \
  -e CLOUDFLARE_ZONE_ID=your_zone_id \
  -e BASIC_AUTH_USERNAME=your_username \
  -e BASIC_AUTH_PASSWORD=your_password \
  timeteus/dyndns-cloudflare-proxy:latest
```

### Using Docker Compose

1. Clone this repository:
```bash
git clone https://github.com/timeteus/dyndns-cloudflare-proxy.git
cd dyndns-cloudfare-proxy
```

2. Copy the example environment file and configure it:
```bash
cp .env.example .env
# Edit .env with your Cloudflare credentials
```

3. Start the service:
```bash
docker-compose up -d
```

### Building from Source

```bash
# Build locally
docker build -t dyndns-cloudflare-proxy .

docker run -d \
  -p 8080:8080 \
  -e CLOUDFLARE_API_TOKEN=your_api_token \
  -e CLOUDFLARE_ZONE_ID=your_zone_id \
  -e BASIC_AUTH_USERNAME=your_username \
  -e BASIC_AUTH_PASSWORD=your_password \
  dyndns-cloudflare-proxy
```

### Running Locally

1. Build the binary:
```bash
go build -o dyndns-cloudflare-proxy
```

2. Set environment variables and run:
```bash
export CLOUDFLARE_API_TOKEN=your_api_token
export CLOUDFLARE_ZONE_ID=your_zone_id
export BASIC_AUTH_USERNAME=your_username
export BASIC_AUTH_PASSWORD=your_password
./dyndns-cloudflare-proxy
```

## Configuration

The service is configured entirely through environment variables:

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `CLOUDFLARE_API_TOKEN` | Yes | - | Your Cloudflare API token (Bearer token) |
| `CLOUDFLARE_ZONE_ID` | Yes | - | The Zone ID for your domain |
| `BASIC_AUTH_USERNAME` | No | - | Username for basic authentication |
| `BASIC_AUTH_PASSWORD` | No | - | Password for basic authentication |
| `PORT` | No | 8080 | Port the server listens on |

### Getting Cloudflare Credentials

1. **API Token**: Log into Cloudflare → My Profile → API Tokens → Create Token
   - Use the "Edit zone DNS" template or create a custom token with `Zone.DNS` edit permissions
2. **Zone ID**: Select your domain → Overview → Zone ID (right sidebar)

## Usage

### DynDNS Update Endpoint

The service exposes the DynDNS update endpoint at `/nic/update`.

#### Request Format

```
GET /nic/update?hostname=subdomain.yourdomain.com&myip=1.2.3.4
```

**Parameters:**
- `hostname` (required): The fully qualified domain name to update
- `myip` (optional): The IP address to set. If not provided, the client's IP is used

**Authentication:**
If `BASIC_AUTH_USERNAME` and `BASIC_AUTH_PASSWORD` are set, requests must include HTTP Basic Authentication.

#### Response Codes

The service returns standard DynDNS response codes:

- `good 1.2.3.4` - Update successful
- `nochg 1.2.3.4` - IP address is already set to this value
- `badauth` - Authentication failed
- `notfqdn` - Hostname parameter is missing or invalid
- `badip` - Invalid IP address
- `911` - Server error (check logs)

### Examples

#### Using curl

```bash
curl -u username:password "http://localhost:8080/nic/update?hostname=home.example.com&myip=1.2.3.4"
```

#### Using wget

```bash
wget --http-user=username --http-password=password \
  -O - "http://localhost:8080/nic/update?hostname=home.example.com&myip=1.2.3.4"
```

#### Auto-detect IP

```bash
curl -u username:password "http://localhost:8080/nic/update?hostname=home.example.com"
```

### Health Check

A health check endpoint is available at `/health`:

```bash
curl http://localhost:8080/health
```

## Integration with DynDNS Clients

This service is compatible with any DynDNS client. Configure your client with:

- **Server/URL**: `http://your-server:8080/nic/update`
- **Username**: Your configured `BASIC_AUTH_USERNAME`
- **Password**: Your configured `BASIC_AUTH_PASSWORD`
- **Hostname**: The FQDN you want to update

## Protocol Reference

This implementation follows the DynDNS protocol specification: https://dyndns.it/note-tecniche/dyndns-it-api/

## License

MIT License

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.