#!/bin/bash
# =============================================================================
# Sub2API Docker Deployment Preparation Script
# =============================================================================
# This script prepares deployment files for Sub2API:
#   - Downloads docker-compose.local.yml and .env.example
#   - Uses the Saksk-IT SubAPI image by default, with SUBAPI_IMAGE override support
#   - Generates secure secrets (JWT_SECRET, TOTP_ENCRYPTION_KEY, POSTGRES_PASSWORD, etc.)
#   - Creates necessary data directories
#   - Optionally configures Caddy HTTPS for a user-provided Cloudflare-backed domain
#
# After running this script, you can start services with:
#   docker compose up -d
# =============================================================================

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# GitHub raw content base URL
GITHUB_RAW_URL="https://raw.githubusercontent.com/Saksk-IT/SubAPI/main/deploy"
DEFAULT_SUBAPI_IMAGE="ghcr.io/saksk-it/subapi:latest"
SUBAPI_IMAGE="${SUBAPI_IMAGE:-$DEFAULT_SUBAPI_IMAGE}"
SUB2API_DOMAIN="${SUB2API_DOMAIN:-${DOMAIN:-}}"
ACME_EMAIL="${ACME_EMAIL:-}"
AUTO_START="${AUTO_START:-false}"

# Print colored message
print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Generate random secret
generate_secret() {
    openssl rand -hex 32
}

# Check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

sed_inplace() {
    local expression="$1"
    local file="$2"

    if sed --version >/dev/null 2>&1; then
        sed -i "$expression" "$file"
    else
        sed -i '' "$expression" "$file"
    fi
}

# Read user input from the terminal even when the script is fetched via curl | bash.
prompt_input() {
    local prompt="$1"
    local reply=""

    if [ -t 0 ]; then
        read -r -p "$prompt" reply
    elif [ -e /dev/tty ]; then
        {
            printf "%s" "$prompt" > /dev/tty
            IFS= read -r reply < /dev/tty || reply=""
        } 2>/dev/null || reply=""
    fi

    printf "%s" "$reply"
}

confirm() {
    local prompt="$1"
    local reply
    reply=$(prompt_input "$prompt")
    [[ $reply =~ ^[Yy]$ ]]
}

is_truthy() {
    case "${1:-}" in
        1|true|TRUE|yes|YES|y|Y) return 0 ;;
        *) return 1 ;;
    esac
}

normalize_domain() {
    local domain="$1"

    domain="${domain#http://}"
    domain="${domain#https://}"
    domain="${domain%%/*}"
    domain="${domain%%:*}"
    domain="${domain//[[:space:]]/}"

    printf "%s" "$domain"
}

validate_domain() {
    local domain="$1"

    if [ -z "$domain" ]; then
        return 0
    fi

    if [[ ! "$domain" =~ ^[A-Za-z0-9.-]+$ ]] || [[ "$domain" != *.* ]] || [[ "$domain" == .* ]] || [[ "$domain" == *. ]]; then
        return 1
    fi

    return 0
}

set_env_var() {
    local key="$1"
    local value="$2"
    local file="${3:-.env}"

    if grep -q "^${key}=" "$file"; then
        sed_inplace "s#^${key}=.*#${key}=${value}#" "$file"
    else
        printf '%s=%s\n' "$key" "$value" >> "$file"
    fi
}

set_env_var_if_empty() {
    local key="$1"
    local value="$2"
    local file="${3:-.env}"

    if ! grep -q "^${key}=.\+" "$file"; then
        set_env_var "$key" "$value" "$file"
    fi
}

get_public_ipv4() {
    if command_exists curl; then
        curl -4fsS https://api.ipify.org 2>/dev/null || curl -4fsS https://ifconfig.me 2>/dev/null || true
    fi
}

configure_compose_image() {
    if grep -q "image: ghcr.io/saksk-it/subapi:latest" docker-compose.yml; then
        sed_inplace 's#image: ghcr.io/saksk-it/subapi:latest#image: ${SUBAPI_IMAGE:-ghcr.io/saksk-it/subapi:latest}#' docker-compose.yml
    fi
}

write_compose_override() {
    local enable_https="$1"

    if [ "$enable_https" = "true" ]; then
        cat > docker-compose.override.yml <<'EOF'
services:
  sub2api:
    env_file:
      - .env

  caddy:
    image: caddy:2-alpine
    container_name: sub2api-caddy
    restart: unless-stopped
    depends_on:
      sub2api:
        condition: service_started
    ports:
      - "80:80"
      - "443:443"
      - "443:443/udp"
    volumes:
      - ./Caddyfile:/etc/caddy/Caddyfile:ro
      - ./caddy_data:/data
      - ./caddy_config:/config
      - ./caddy_logs:/var/log/caddy
    networks:
      - sub2api-network
    logging:
      driver: json-file
      options:
        max-size: "20m"
        max-file: "5"
EOF
    else
        cat > docker-compose.override.yml <<'EOF'
services:
  sub2api:
    env_file:
      - .env
EOF
    fi
}

write_caddyfile() {
    local domain="$1"
    local email="$2"

    cat > Caddyfile <<EOF
{
	email ${email}
}

${domain} {
	encode {
		zstd
		gzip 6
		minimum_length 256
	}

	request_body {
		max_size 256MB
	}

	header {
		Strict-Transport-Security "max-age=31536000"
		X-Content-Type-Options "nosniff"
		Referrer-Policy "no-referrer-when-downgrade"
	}

	reverse_proxy sub2api:8080 {
		health_uri /health
		health_interval 30s
		health_timeout 10s
		health_status 200

		header_up X-Real-IP {remote_host}
		header_up CF-Connecting-IP {http.request.header.CF-Connecting-IP}

		transport http {
			versions h2c 1.1
			keepalive 120s
			keepalive_idle_conns 256
			read_buffer 16KB
			write_buffer 16KB
			compression off
		}

		fail_duration 30s
		max_fails 3
		unhealthy_status 500 502 503 504
	}

	log {
		output file /var/log/caddy/sub2api.log {
			roll_size 50mb
			roll_keep 10
			roll_keep_for 720h
		}
		format json
		level INFO
	}

	handle_errors {
		respond "{err.status_code} {err.status_text}"
	}
}
EOF
}

maybe_start_services() {
    if ! is_truthy "$AUTO_START"; then
        return 0
    fi

    if ! command_exists docker; then
        print_warning "AUTO_START=true was set, but Docker is not installed or not in PATH. Skipping startup."
        return 0
    fi

    if ! docker compose version >/dev/null 2>&1; then
        print_warning "AUTO_START=true was set, but Docker Compose v2 is unavailable. Skipping startup."
        return 0
    fi

    print_info "Validating Docker Compose configuration..."
    docker compose config --quiet

    print_info "Pulling images..."
    docker compose pull

    print_info "Starting services..."
    docker compose up -d
    docker compose ps
}

# Main installation function
main() {
    echo ""
    echo "=========================================="
    echo "  Sub2API Deployment Preparation"
    echo "=========================================="
    echo ""

    # Check if openssl is available
    if ! command_exists openssl; then
        print_error "openssl is not installed. Please install openssl first."
        exit 1
    fi

    SUB2API_DOMAIN=$(normalize_domain "$SUB2API_DOMAIN")
    if [ -z "$SUB2API_DOMAIN" ]; then
        SUB2API_DOMAIN=$(normalize_domain "$(prompt_input "Enter domain for HTTPS (leave blank to use IP:port only): ")")
    fi

    if ! validate_domain "$SUB2API_DOMAIN"; then
        print_error "Invalid domain: ${SUB2API_DOMAIN}"
        print_error "Use a hostname such as api.example.com, without http:// or path."
        exit 1
    fi

    if [ -n "$SUB2API_DOMAIN" ] && [ -z "$ACME_EMAIL" ]; then
        ACME_EMAIL="admin@${SUB2API_DOMAIN}"
    fi

    # Check if deployment already exists
    if [ -f "docker-compose.yml" ] && [ -f ".env" ]; then
        print_warning "Deployment files already exist in current directory."
        if ! confirm "Overwrite existing docker-compose.yml and .env? (y/N): "; then
            print_info "Cancelled."
            exit 0
        fi
    fi

    # Download docker-compose.local.yml and save as docker-compose.yml
    print_info "Downloading docker-compose.yml..."
    if command_exists curl; then
        curl -sSL "${GITHUB_RAW_URL}/docker-compose.local.yml" -o docker-compose.yml
    elif command_exists wget; then
        wget -q "${GITHUB_RAW_URL}/docker-compose.local.yml" -O docker-compose.yml
    else
        print_error "Neither curl nor wget is installed. Please install one of them."
        exit 1
    fi
    print_success "Downloaded docker-compose.yml"
    configure_compose_image

    # Download .env.example
    print_info "Downloading .env.example..."
    if command_exists curl; then
        curl -sSL "${GITHUB_RAW_URL}/.env.example" -o .env.example
    else
        wget -q "${GITHUB_RAW_URL}/.env.example" -O .env.example
    fi
    print_success "Downloaded .env.example"

    # Generate .env file with auto-generated secrets
    print_info "Generating secure secrets..."
    echo ""

    # Generate secrets
    JWT_SECRET=$(generate_secret)
    TOTP_ENCRYPTION_KEY=$(generate_secret)
    POSTGRES_PASSWORD=$(generate_secret)
    REDIS_PASSWORD=$(generate_secret)
    ADMIN_PASSWORD=$(openssl rand -base64 24 | tr -d '\n')

    # Create .env from .env.example
    cp .env.example .env

    set_env_var "SUBAPI_IMAGE" "$SUBAPI_IMAGE"
    set_env_var "JWT_SECRET" "$JWT_SECRET"
    set_env_var "TOTP_ENCRYPTION_KEY" "$TOTP_ENCRYPTION_KEY"
    set_env_var "POSTGRES_PASSWORD" "$POSTGRES_PASSWORD"
    set_env_var "REDIS_PASSWORD" "$REDIS_PASSWORD"
    set_env_var "ADMIN_PASSWORD" "$ADMIN_PASSWORD"
    set_env_var_if_empty "ADMIN_EMAIL" "admin@sub2api.local"

    if [ -n "$SUB2API_DOMAIN" ]; then
        set_env_var "BIND_HOST" "127.0.0.1"
        set_env_var "SERVER_FRONTEND_URL" "https://${SUB2API_DOMAIN}"
        set_env_var "SECURITY_URL_ALLOWLIST_ALLOW_INSECURE_HTTP" "false"
        set_env_var "SECURITY_URL_ALLOWLIST_ALLOW_PRIVATE_HOSTS" "false"
    fi

    # Create data directories
    print_info "Creating data directories..."
    mkdir -p data postgres_data redis_data
    if [ -n "$SUB2API_DOMAIN" ]; then
        mkdir -p caddy_data caddy_config caddy_logs
        write_caddyfile "$SUB2API_DOMAIN" "$ACME_EMAIL"
        write_compose_override "true"
    else
        write_compose_override "false"
    fi
    print_success "Created data directories"

    # Set secure permissions for .env file (readable/writable only by owner)
    chmod 600 .env
    echo ""

    # Display completion message
    echo "=========================================="
    echo "  Preparation Complete!"
    echo "=========================================="
    echo ""
    echo "Generated secure credentials:"
    echo "  POSTGRES_PASSWORD:     ${POSTGRES_PASSWORD}"
    echo "  REDIS_PASSWORD:        ${REDIS_PASSWORD}"
    echo "  ADMIN_EMAIL:           $(grep '^ADMIN_EMAIL=' .env | cut -d= -f2-)"
    echo "  ADMIN_PASSWORD:        ${ADMIN_PASSWORD}"
    echo "  JWT_SECRET:            ${JWT_SECRET}"
    echo "  TOTP_ENCRYPTION_KEY:   ${TOTP_ENCRYPTION_KEY}"
    echo "  SUBAPI_IMAGE:          ${SUBAPI_IMAGE}"
    echo ""
    print_warning "These credentials have been saved to .env file."
    print_warning "Please keep them secure and do not share publicly!"
    echo ""
    echo "Directory structure:"
    echo "  docker-compose.yml        - Docker Compose configuration"
    echo "  docker-compose.override.yml - Compose overrides (env_file and optional Caddy)"
    echo "  .env                      - Environment variables (generated secrets)"
    echo "  .env.example              - Example template (for reference)"
    echo "  data/                     - Application data (will be created on first run)"
    echo "  postgres_data/            - PostgreSQL data"
    echo "  redis_data/               - Redis data"
    if [ -n "$SUB2API_DOMAIN" ]; then
        echo "  Caddyfile                 - HTTPS reverse proxy configuration"
        echo "  caddy_data/               - Caddy certificates and ACME state"
        echo "  caddy_config/             - Caddy runtime config"
        echo "  caddy_logs/               - Caddy access logs"
    fi
    echo ""
    if [ -n "$SUB2API_DOMAIN" ]; then
        PUBLIC_IPV4=$(get_public_ipv4)
        echo "Cloudflare DNS:"
        if [ -n "$PUBLIC_IPV4" ]; then
            echo "  Add an A record: ${SUB2API_DOMAIN} -> ${PUBLIC_IPV4}"
        else
            echo "  Add an A record: ${SUB2API_DOMAIN} -> this server's public IPv4"
        fi
        echo "  Set SSL/TLS mode to: Full (strict)"
        echo "  Do NOT use: Flexible"
        echo ""
    fi
    echo "Next steps:"
    echo "  1. (Optional) Edit .env to customize configuration"
    echo "  2. Start services:"
    echo "     docker compose up -d"
    echo ""
    echo "  3. View logs:"
    echo "     docker compose logs -f sub2api"
    if [ -n "$SUB2API_DOMAIN" ]; then
        echo "     docker compose logs -f caddy"
    fi
    echo ""
    echo "  4. Access Web UI:"
    if [ -n "$SUB2API_DOMAIN" ]; then
        echo "     https://${SUB2API_DOMAIN}"
        echo ""
        echo "  5. Verify:"
        echo "     curl -fsS http://127.0.0.1:8080/health"
        echo "     curl -fsS https://${SUB2API_DOMAIN}/health"
    else
        echo "     http://localhost:8080"
    fi
    echo ""
    maybe_start_services
    echo ""
}

# Run main function
main "$@"
