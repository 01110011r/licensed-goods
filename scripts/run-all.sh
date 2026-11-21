#!/usr/bin/env bash
#
# Runs the whole stack with docker compose.
#
#   scripts/run-all.sh          start test databases, run Go and web checks,
#                               then start the API and web app if they pass
#   scripts/run-all.sh test     only the test databases and checks
#   scripts/run-all.sh up       only the app (database, migrations, API, web)
#   scripts/run-all.sh down     stop everything (keeps the dev database volume)
#   scripts/run-all.sh reset    stop everything and delete all volumes
#
# REVIEWER_TOKEN is read from the environment, then from .env. If neither sets
# it, a random token is generated for this run and printed.

set -euo pipefail

cd "$(dirname "$0")/.."

COMPOSE=(docker compose)

log() { printf '\n==> %s\n' "$*"; }
die() { printf 'error: %s\n' "$*" >&2; exit 1; }

require_docker() {
  command -v docker >/dev/null 2>&1 || die "docker is not installed"
  docker compose version >/dev/null 2>&1 || die "docker compose v2 is required"
  docker info >/dev/null 2>&1 || die "the docker daemon is not running (or you lack permission)"
}

load_reviewer_token() {
  if [[ -n "${REVIEWER_TOKEN:-}" ]]; then
    return
  fi

  if [[ -f .env ]]; then
    local from_file
    from_file="$(sed -n 's/^REVIEWER_TOKEN=//p' .env | tail -n 1)"
    if [[ -n "$from_file" && "$from_file" != "change-me" ]]; then
      REVIEWER_TOKEN="$from_file"
      export REVIEWER_TOKEN
      return
    fi
  fi

  REVIEWER_TOKEN="$(head -c 24 /dev/urandom | base64 | tr -dc 'A-Za-z0-9' | head -c 32)"
  export REVIEWER_TOKEN
  echo "No REVIEWER_TOKEN set; generated one for this run: $REVIEWER_TOKEN"
}

run_tests() {
  local failed=0

  log "Go: vet and tests (against the test database)"
  "${COMPOSE[@]}" --profile test run --rm --build test-go || failed=1

  log "Web: lint and production build"
  "${COMPOSE[@]}" --profile test run --rm --build test-web || failed=1

  if (( failed )); then
    die "tests failed; the app was not started"
  fi
  log "All checks passed"
}

start_app() {
  log "Starting database, migrations, API and web"
  "${COMPOSE[@]}" up -d --build --wait api web

  printf '\nStack is running:\n'
  printf '  Web app   http://localhost:%s\n' "${WEB_PORT:-5173}"
  printf '  API       http://localhost:%s/health\n' "${API_PORT:-8080}"
  printf '  Database  localhost:%s (user postgres, database licensed_goods)\n' "${DB_PORT:-5433}"
  printf '\nReviewer token for this run: %s\n' "$REVIEWER_TOKEN"
  printf 'Enter it in the web app when opening Licence reviews or Settings.\n'
  printf '\nStop with: scripts/run-all.sh down\n'
}

main() {
  local command="${1:-all}"
  require_docker

  case "$command" in
    all)
      load_reviewer_token
      log "Starting test databases"
      "${COMPOSE[@]}" --profile test up -d --wait db-test
      run_tests
      start_app
      ;;
    test)
      load_reviewer_token
      log "Starting test databases"
      "${COMPOSE[@]}" --profile test up -d --wait db-test
      run_tests
      ;;
    up)
      load_reviewer_token
      start_app
      ;;
    down)
      "${COMPOSE[@]}" --profile test down
      ;;
    reset)
      "${COMPOSE[@]}" --profile test down -v
      ;;
    *)
      die "unknown command '$command' (use all, test, up, down or reset)"
      ;;
  esac
}

main "$@"
