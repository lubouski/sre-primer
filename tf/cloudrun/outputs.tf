output "cloud_sql_instance" {
  value = data.google_sql_database_instance.cloud_sql_instance.private_ip_address
}

output "cloud_sql_secret_id" {
  value = data.google_secret_manager_secret.cloud_sql_secret.id
}

output "cloud_sql_connection_name" {
    value = data.google_sql_database_instance.cloud_sql_instance.connection_name
}

output "cloud_sql_connector_id" {
  value = data.google_vpc_access_connector.cloud_sql_connector.id
}

output "cloud_run_service_account_email" {
  value = data.google_service_account.cloud_run_service_account.email
}

output "cloud_run_service_url" {
    description = "The URL of the Cloud Run service"
    value = google_cloud_run_v2_service.cloud_run_service.uri
}
