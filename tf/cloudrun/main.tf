data "google_sql_database_instance" "cloud_sql_instance" {
  name = var.cloud_sql_instance_name
  project = var.project_id
}

data "google_secret_manager_secret" "cloud_sql_secret" {
  secret_id = var.secret_id
}

data "google_vpc_access_connector" "cloud_sql_connector" {
  name = var.vpc_access_connector_name
}

data "google_service_account" "cloud_run_service_account" {
  account_id = var.cloud_run_sa_name
}

# --- Cloud Run ---
resource "google_project_iam_member" "cloud_run_sql_client" {
  project = var.project_id
  role    = "roles/cloudsql.client"
  member  = "serviceAccount:${data.google_service_account.cloud_run_service_account.email}"
}

resource "google_cloud_run_v2_service" "cloud_run_service" {
  name = var.cloud_run_service_name
  location = var.region
  ingress = "INGRESS_TRAFFIC_ALL"

  deletion_protection = false
  template {
    containers {
      image = var.cloud_run_image

      resources {
        limits = {
          cpu    = "1"
          memory = "1024Mi"
        }
      }

      startup_probe {
        failure_threshold     = 5
        initial_delay_seconds = 10
        timeout_seconds       = 3
        period_seconds        = 3

        http_get {
          path = "/health"
        }
      }
      liveness_probe {
        failure_threshold     = 5
        initial_delay_seconds = 10
        timeout_seconds       = 3
        period_seconds        = 3

        http_get {
          path = "/readiness"
        }
      }
      ports {
        container_port = var.cloud_run_container_port
      }
      volume_mounts {
        mount_path = "/cloudsql"
        name       = "cloudsql"
      }
      env {
        name = "DB_PASSWORD"
        value_source {
          secret_key_ref {
            secret = data.google_secret_manager_secret.cloud_sql_secret.id
            version = "latest"
          }
        }
      }
      env {
        name = "DB_HOST"
        value = data.google_sql_database_instance.cloud_sql_instance.private_ip_address
      }
      env {
        name = "DB_USER"
        value = var.db_user
      }
      env {
        name = "DB_NAME"
        value = var.db_name
      }
    }
    volumes {
      name = "cloudsql"
      cloud_sql_instance {
        instances = [data.google_sql_database_instance.cloud_sql_instance.connection_name]
      }
    }
    scaling {
      max_instance_count = 5
    }
    vpc_access {
      connector = data.google_vpc_access_connector.cloud_sql_connector.id
      egress = "ALL_TRAFFIC"
    }
    service_account = data.google_service_account.cloud_run_service_account.email
  }
  depends_on = [
    google_project_iam_member.cloud_run_sql_client,
  ]
}

resource "google_cloud_run_service_iam_binding" "default_access_binding" {
  location = google_cloud_run_v2_service.cloud_run_service.location
  service  = google_cloud_run_v2_service.cloud_run_service.name
  role     = "roles/run.invoker"
  members = [
    "allUsers"
  ]
}
