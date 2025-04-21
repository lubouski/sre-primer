variable "project_id" {
  description = "The GCP project ID."
  type = string
}

variable "region" {
  description = "The GCP region for Cloud SQL"
  type = string
  default = "europe-west3"
}

# --- Cloud SQL ---
variable "cloud_sql_instance_name" {
  description = "The name of the Cloud SQL instance"
  type = string
  default = "hello-service-db-instance"
}

variable "db_name" {
  description = "The name of the database to create"
  type = string
  default = "hello-service-db"
}

variable "db_user" {
  description = "The name of the database user"
  type = string
  default = "hello-service-db-user"
}

variable "vpc_access_connector_name" {
    description = "value of vpc_access_connector_name variable"
    type = string
    default = "tf-cloud-sql-vpc-con"
}

# --- Secret Manager ---
variable "secret_id" {
  description = "The ID for the Secret Manager secret holding DB password"
  type = string
  default = "hello-service-db-password"
}

# -- Cloud Run ---
variable "cloud_run_sa_name" {
    description = "The name for the Cloud Run service"
    type = string
    default = "cloud-run-service-account"
}

variable "cloud_run_service_name" {
  description = "The name for the Cloud Run service"
  type = string
  default = "hello-service"
}

variable "cloud_run_image" {
  description = "Container Image"
  type = string
  default = "lubowsky/hello-server:server"
}

variable "cloud_run_container_port" {
  description = "The port your container listens on"
  type = string
  default = "8080"
}
