variable "project_id" {
  description = "The GCP project ID."
  type = string
}

variable "region" {
  description = "The GCP region for Cloud SQL"
  type = string
  default = "europe-west3"
}

variable "zone" {
  description = "The GCP zone for GCE migrations vm"
  type = string
  default = "europe-west3-b"
}

variable "cloud_sql_instance_name" {
  description = "The name of the Cloud SQL instance"
  type = string
  default = "hello-service-db-instance"
}

variable "db_version" {
  description = "The Postgres version for the Cloud SQL instance"
  type = string
  default = "POSTGRES_17"
}

variable "tb_tier" {
  description = "The machine tier for the Cloud SQL instance"
  type = string
  default = "db-custom-1-3840"
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

variable "enable_public_ip" {
  description = "Set to true to enable public IP"
  type = bool
  default = false
}

variable "cloud_sql_private_subnet_name" {
  description = "The name for the subnet within the private network"
  type = string
  default = "tf-sql-private-network"
}

variable "cloud_sql_vpc_connector_private_subnet_name" {
  description = "The name for the subnet within the private network"
  type = string
  default = "tf-sql-vpc-connector-private-network"
}

variable "vpc_access_connector_name" {
    description = "value of vpc_access_connector_name variable"
    type = string
    default = "tf-cloud-sql-vpc-con"
}

variable "cloud_sql_subnet_cidr_range" {
  description = "The IP CIDR range for the subnet. Must not overlap with Private Service Access range"
  type = string
  default = "10.1.0.0/24"
}

variable "cloud_sql_vpc_connector_subnet_cidr_range" {
  description = "The IP CIDR range for the subnet. Must not overlap with Private Service Access range"
  type = string
  default = "10.2.0.0/28"
}

variable "private_service_access_cidr_range" {
  description = "The prefix length for the IP range for Private Service Access (e.g 16,20,24)"
  type = string
  default = "16"
}

# --- Secret Manager ---
variable "secret_id" {
  description = "The ID for the Secret Manager secret holding DB password"
  type = string
  default = "hello-service-db-password"
}
