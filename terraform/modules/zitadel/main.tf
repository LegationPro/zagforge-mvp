# --- Zitadel Identity Provider (Cloud Run) ---
# Self-hosted Zitadel instance for authentication, SSO, and org management.
# Zitadel stores its data in its own Postgres database (separate from the app DB).

resource "google_service_account" "zitadel" {
  account_id   = "${var.name_prefix}-zitadel"
  display_name = "Zagforge Zitadel service account"
}

resource "google_cloud_run_v2_service" "zitadel" {
  name     = "${var.name_prefix}-zitadel"
  location = var.region
  ingress  = "INGRESS_TRAFFIC_ALL"

  template {
    service_account = google_service_account.zitadel.email

    scaling {
      min_instance_count = var.min_instances
      max_instance_count = var.max_instances
    }

    containers {
      # Zitadel official image — pinned to a stable release.
      # Update this when upgrading Zitadel versions.
      image = var.zitadel_image

      ports {
        container_port = 8080
      }

      # --- Zitadel configuration via env vars ---
      # See: https://zitadel.com/docs/self-hosting/manage/configure
      env {
        name  = "ZITADEL_PORT"
        value = "8080"
      }
      env {
        name  = "ZITADEL_EXTERNALDOMAIN"
        value = var.external_domain
      }
      env {
        name  = "ZITADEL_EXTERNALPORT"
        value = "443"
      }
      env {
        name  = "ZITADEL_EXTERNALSECURE"
        value = "true"
      }
      env {
        name  = "ZITADEL_TLS_ENABLED"
        value = "false" # Cloud Run terminates TLS
      }

      # --- Database connection ---
      # Zitadel uses its own Postgres database (not the app database).
      # Connection string injected via Doppler at deploy time:
      #   ZITADEL_DATABASE_POSTGRES_HOST
      #   ZITADEL_DATABASE_POSTGRES_PORT
      #   ZITADEL_DATABASE_POSTGRES_DATABASE
      #   ZITADEL_DATABASE_POSTGRES_USER_USERNAME
      #   ZITADEL_DATABASE_POSTGRES_USER_PASSWORD
      #   ZITADEL_DATABASE_POSTGRES_USER_SSL_MODE
      #   ZITADEL_DATABASE_POSTGRES_ADMIN_USERNAME
      #   ZITADEL_DATABASE_POSTGRES_ADMIN_PASSWORD
      #   ZITADEL_DATABASE_POSTGRES_ADMIN_SSL_MODE

      # --- Master key (Doppler-managed) ---
      # ZITADEL_MASTERKEY — 32-byte key for encrypting secrets at rest.
      # Injected via: doppler run -- gcloud run services update ...

      resources {
        limits = {
          cpu    = var.cpu
          memory = var.memory
        }
      }

      startup_probe {
        http_get {
          path = "/debug/healthz"
        }
        initial_delay_seconds = 10
        period_seconds        = 5
        failure_threshold     = 20
      }

      liveness_probe {
        http_get {
          path = "/debug/healthz"
        }
        period_seconds = 30
      }
    }
  }

  lifecycle {
    ignore_changes = [
      template[0].containers[0].env,
    ]
  }
}

# Zitadel must be publicly accessible (login pages, OIDC endpoints).
resource "google_cloud_run_v2_service_iam_member" "public" {
  name     = google_cloud_run_v2_service.zitadel.name
  location = var.region
  role     = "roles/run.invoker"
  member   = "allUsers"
}
