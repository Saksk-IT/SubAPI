#!/bin/bash
# =============================================================================
# Sub2API Docker Deployment Preparation Script
# =============================================================================
# This script prepares deployment files for Sub2API:
#   - Updates server packages and installs Docker Engine when needed
#   - Downloads docker-compose.local.yml and .env.example
#   - Uses the Saksk-IT SubAPI image by default, with SUBAPI_IMAGE override support
#   - Generates secure secrets (JWT_SECRET, TOTP_ENCRYPTION_KEY, POSTGRES_PASSWORD, etc.)
#   - Creates necessary data directories
#   - Optionally configures Caddy HTTPS for a user-provided Cloudflare-backed domain
#   - Starts Docker Compose services by default
#
# Typical usage:
#   curl -fsSL https://raw.githubusercontent.com/Saksk-IT/SubAPI/main/deploy/docker-deploy.sh | sudo bash
# =============================================================================

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# GitHub raw content base URL
GITHUB_RAW_URL="${GITHUB_RAW_URL:-https://raw.githubusercontent.com/Saksk-IT/SubAPI/main/deploy}"
DEFAULT_SUBAPI_IMAGE="ghcr.io/saksk-it/subapi:latest"
SUBAPI_IMAGE="${SUBAPI_IMAGE:-$DEFAULT_SUBAPI_IMAGE}"
SUB2API_DOMAIN="${SUB2API_DOMAIN:-${DOMAIN:-}}"
ACME_EMAIL="${ACME_EMAIL:-}"
DEPLOY_DIR="${DEPLOY_DIR:-/opt/sub2api}"
INSTALL_DOCKER="${INSTALL_DOCKER:-auto}"
SYSTEM_UPGRADE="${SYSTEM_UPGRADE:-true}"
AUTO_START="${AUTO_START:-true}"
CONFIGURE_UFW="${CONFIGURE_UFW:-true}"
REQUIRE_HEALTHCHECK="${REQUIRE_HEALTHCHECK:-true}"
ALLOW_NON_ROOT="${ALLOW_NON_ROOT:-false}"

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

is_falsey() {
    case "${1:-}" in
        0|false|FALSE|no|NO|n|N) return 0 ;;
        *) return 1 ;;
    esac
}

require_root() {
    if is_truthy "$ALLOW_NON_ROOT"; then
        return 0
    fi

    if [ "$(id -u)" -ne 0 ]; then
        print_error "This full deployment script must run as root."
        print_error "Use: curl -fsSL ${GITHUB_RAW_URL}/docker-deploy.sh | sudo bash"
        print_error "Or set ALLOW_NON_ROOT=true INSTALL_DOCKER=false DEPLOY_DIR=/path for local generation tests."
        exit 1
    fi
}

ensure_ubuntu_or_debian() {
    if [ ! -r /etc/os-release ]; then
        print_error "/etc/os-release not found. Automatic Docker installation is only supported on Ubuntu/Debian."
        exit 1
    fi

    # shellcheck disable=SC1091
    . /etc/os-release

    case "${ID:-}" in
        ubuntu|debian) ;;
        *)
            print_error "Unsupported OS for automatic Docker installation: ${ID:-unknown}"
            print_error "Install Docker manually, then rerun with INSTALL_DOCKER=false."
            exit 1
            ;;
    esac

    if [ -z "${VERSION_CODENAME:-}" ]; then
        print_error "VERSION_CODENAME is empty in /etc/os-release. Cannot configure Docker apt repository."
        exit 1
    fi
}

docker_compose_available() {
    command_exists docker && docker compose version >/dev/null 2>&1
}

install_docker_if_needed() {
    if is_falsey "$INSTALL_DOCKER"; then
        print_info "INSTALL_DOCKER=false, skipping Docker installation."
        return 0
    fi

    if ! command_exists apt-get; then
        if docker_compose_available; then
            print_success "Docker Compose is already available."
            return 0
        fi

        print_error "apt-get is unavailable and Docker Compose is not installed."
        print_error "Install Docker manually, then rerun with INSTALL_DOCKER=false."
        exit 1
    fi

    ensure_ubuntu_or_debian

    export DEBIAN_FRONTEND="${DEBIAN_FRONTEND:-noninteractive}"
    export NEEDRESTART_MODE="${NEEDRESTART_MODE:-a}"

    print_info "Updating apt package index..."
    apt-get update

    if is_truthy "$SYSTEM_UPGRADE"; then
        print_info "Upgrading installed packages..."
        apt-get upgrade -y
    else
        print_info "SYSTEM_UPGRADE=false, skipping package upgrade."
    fi

    print_info "Installing base packages..."
    apt-get install -y ca-certificates curl openssl

    if docker_compose_available; then
        print_success "Docker Compose is already available."
        if command_exists systemctl; then
            systemctl enable --now docker >/dev/null 2>&1 || true
        fi
        return 0
    fi

    print_info "Installing Docker Engine and Docker Compose plugin..."
    for pkg in docker.io docker-doc docker-compose docker-compose-v2 podman-docker containerd runc; do
        apt-get remove -y "$pkg" >/dev/null 2>&1 || true
    done

    install -m 0755 -d /etc/apt/keyrings
    curl --connect-timeout 10 --max-time 30 --retry 3 --retry-delay 2 -fsSL "https://download.docker.com/linux/${ID}/gpg" -o /etc/apt/keyrings/docker.asc
    chmod a+r /etc/apt/keyrings/docker.asc

    cat > /etc/apt/sources.list.d/docker.list <<EOF
deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/${ID} ${VERSION_CODENAME} stable
EOF

    apt-get update
    apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin

    if command_exists systemctl; then
        systemctl enable --now docker
    fi

    if ! docker_compose_available; then
        print_error "Docker Compose installation did not complete successfully."
        exit 1
    fi

    print_success "Docker Engine and Docker Compose plugin are ready."
}

prepare_deploy_dir() {
    print_info "Preparing deployment directory: ${DEPLOY_DIR}"
    mkdir -p "$DEPLOY_DIR"
    cd "$DEPLOY_DIR"
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
        curl --max-time 5 -4fsS https://api.ipify.org 2>/dev/null || curl --max-time 5 -4fsS https://ifconfig.me 2>/dev/null || true
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
    verify_deployment
}

configure_firewall() {
    local enable_https="$1"

    if ! is_truthy "$CONFIGURE_UFW"; then
        print_info "CONFIGURE_UFW=false, skipping UFW configuration."
        return 0
    fi

    if ! command_exists ufw; then
        if command_exists apt-get; then
            apt-get install -y ufw || {
                print_warning "Failed to install ufw. Please open ports manually."
                return 0
            }
        else
            print_warning "ufw is not installed. Please open ports manually."
            return 0
        fi
    fi

    print_info "Configuring UFW firewall rules..."
    ufw allow OpenSSH >/dev/null 2>&1 || true

    if [ "$enable_https" = "true" ]; then
        ufw allow 80/tcp >/dev/null 2>&1 || true
        ufw allow 443/tcp >/dev/null 2>&1 || true
        ufw delete allow 8080/tcp >/dev/null 2>&1 || true
    else
        ufw allow 8080/tcp >/dev/null 2>&1 || true
    fi

    ufw --force enable >/dev/null 2>&1 || {
        print_warning "Failed to enable ufw. Please open ports manually."
        return 0
    }

    ufw status verbose || true
}

verify_deployment() {
    print_info "Checking local SubAPI health..."
    local i
    local local_ok="false"
    local failed="false"

    for i in $(seq 1 30); do
        if curl --max-time 5 -fsS http://127.0.0.1:8080/health >/dev/null 2>&1; then
            local_ok="true"
            break
        fi
        sleep 2
    done

    if [ "$local_ok" = "true" ]; then
        print_success "Local health check passed: http://127.0.0.1:8080/health"
    else
        failed="true"
        print_warning "Local health check did not pass yet. Inspect logs with: docker compose logs -f sub2api"
    fi

    if [ -z "$SUB2API_DOMAIN" ]; then
        if [ "$failed" = "true" ] && is_truthy "$REQUIRE_HEALTHCHECK"; then
            print_error "Deployment started, but health verification did not complete."
            return 1
        fi
        return 0
    fi

    print_info "Checking HTTPS health for https://${SUB2API_DOMAIN}/health ..."
    local https_ok="false"

    for i in $(seq 1 36); do
        if curl --max-time 10 -fsS "https://${SUB2API_DOMAIN}/health" >/dev/null 2>&1; then
            https_ok="true"
            break
        fi
        sleep 5
    done

    if [ "$https_ok" = "true" ]; then
        print_success "HTTPS health check passed: https://${SUB2API_DOMAIN}/health"
    else
        failed="true"
        print_warning "HTTPS health check did not pass within the wait window."
        print_warning "Confirm Cloudflare A record points to this server, SSL/TLS mode is Full (strict), and ports 80/443 are open."
        print_warning "Recent Caddy logs:"
        docker compose logs --tail=120 caddy || true
    fi

    if [ "$failed" = "true" ] && is_truthy "$REQUIRE_HEALTHCHECK"; then
        print_error "Deployment started, but health verification did not complete."
        print_error "Set REQUIRE_HEALTHCHECK=false only if you intentionally want to skip this completion gate."
        return 1
    fi
}

# Main installation function
main() {
    echo ""
    echo "=========================================="
    echo "  Sub2API Deployment Preparation"
    echo "=========================================="
    echo ""

    require_root
    install_docker_if_needed

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

    prepare_deploy_dir

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
        curl --connect-timeout 10 --max-time 30 --retry 3 --retry-delay 2 -fsSL "${GITHUB_RAW_URL}/docker-compose.local.yml" -o docker-compose.yml
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
        curl --connect-timeout 10 --max-time 30 --retry 3 --retry-delay 2 -fsSL "${GITHUB_RAW_URL}/.env.example" -o .env.example
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
        configure_firewall "true"
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
    if [ -z "$SUB2API_DOMAIN" ]; then
        configure_firewall "false"
    fi
    echo "Next steps:"
    echo "  1. Deployment directory:"
    echo "     ${DEPLOY_DIR}"
    echo ""
    if is_truthy "$AUTO_START"; then
        echo "  2. Services are starting automatically."
        echo "     To restart manually: docker compose up -d"
    else
        echo "  2. Start services manually:"
        echo "     docker compose up -d"
    fi
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
