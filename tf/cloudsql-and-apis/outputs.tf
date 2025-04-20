output "instance_name" {
    description = "THe name of the Cloud SQL instance"
    value = google_sql_database_instance.cloud_sql_instance.name 
}

output "instance_connection_name" {
    description = "The connection name of the instance (used by Cloud SQL Proxy)"
    value = google_sql_database_instance.cloud_sql_instance.connection_name
}

output "instance_private_ip_address" {
    description = "The private IP address of the Cloud SQL instance"
    value = google_sql_database_instance.cloud_sql_instance.private_ip_address
}

output "databese_name" {
    description = "The name of the created database"
    value = google_sql_database.cloud_sql_database.name
}

output "database_user" {
    description = "The name of the created database user"
    value = google_sql_user.cloud_sql_user.name
}

output "user_password" {
  description = "The generated password for the database user"
  value = random_password.db_password.result
  sensitive = true
}

output "secret_id_used" {
    description = "value of secret_id variable"
    value = google_secret_manager_secret.cloud_sql_secret.secret_id
}

output "secret_version_id_used" {
    description = "value of secret_version_id variable"
    value = google_secret_manager_secret_version.cloud_sql_secret_version.id
    sensitive = true
}
