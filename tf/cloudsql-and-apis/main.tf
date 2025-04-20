# -- Cloud Run VPC access
resource "google_project_service" "vpcaccess_api" {
  service = "vpcaccess.googleapis.com"
  disable_on_destroy = false
}

resource "google_compute_network" "cloud_sql_private_network" {
  name = var.cloud_sql_private_subnet_name
  project = var.project_id
  auto_create_subnetworks = false
  delete_default_routes_on_create = false
}

resource "google_compute_subnetwork" "cloud_sql_private_subnet" {
    name = var.cloud_sql_private_subnet_name
    project = var.project_id
    region = var.region
    ip_cidr_range = var.cloud_sql_subnet_cidr_range
    network = google_compute_network.cloud_sql_private_network.id

    private_ip_google_access = true # Allows VMs in this subnet to reach Google API's without external IPs
    
    depends_on = [ google_compute_network.cloud_sql_private_network ]
}

resource "google_compute_subnetwork" "cloud_sql_vpc_connector_private_subnet" {
    name = var.cloud_sql_vpc_connector_private_subnet_name
    project = var.project_id
    region = var.region
    ip_cidr_range = var.cloud_sql_vpc_connector_subnet_cidr_range
    network = google_compute_network.cloud_sql_private_network.id

    private_ip_google_access = true # Allows VMs in this subnet to reach Google API's without external IPs
    
    depends_on = [ google_compute_network.cloud_sql_private_network ]
}

resource "google_compute_global_address" "cloud_sql_private_ip_address" {
    name = "tf-sql-private-ip-address"
    purpose = "VPC_PEERING"
    address_type = "INTERNAL"
    prefix_length = var.private_service_access_cidr_range
    network = google_compute_network.cloud_sql_private_network.id
    depends_on = [ google_compute_network.cloud_sql_private_network ]
}

resource "google_service_networking_connection" "cloud_sql_private_vpc_connection" {
  network = google_compute_network.cloud_sql_private_network.id
  service = "servicenetworking.googleapis.com"
  reserved_peering_ranges = [ google_compute_global_address.cloud_sql_private_ip_address.name ]
  depends_on = [ google_compute_global_address.cloud_sql_private_ip_address ]
}

resource "google_vpc_access_connector" "cloud_sql_connector" {
  name          = var.vpc_access_connector_name
  subnet {
    name = google_compute_subnetwork.cloud_sql_vpc_connector_private_subnet.name
  }
  min_instances = 2
  max_instances = 3
  depends_on = [ google_compute_subnetwork.cloud_sql_vpc_connector_private_subnet ]
}

resource "random_password" "db_password" {
    length = 20
    special = true
    override_special = "_%@" # Limit special chars by SQL    
}

# --- Cloud SQL Instance ---\
resource "google_sql_database_instance" "cloud_sql_instance" {
    name = var.cloud_sql_instance_name
    region = var.region
    database_version = var.db_version
    project = var.project_id

    settings {
        tier = var.tb_tier
        edition = "ENTERPRISE"
        
        ip_configuration {
            ipv4_enabled = var.enable_public_ip
            private_network = google_compute_network.cloud_sql_private_network.id
        }

        backup_configuration {
          enabled = true
        }

        availability_type = "ZONAL"

        disk_autoresize = true
        disk_size = 10 # GB
        disk_type = "PD_SSD"
    }
    deletion_protection = false

    depends_on = [ 
      google_service_networking_connection.cloud_sql_private_vpc_connection,
      google_vpc_access_connector.cloud_sql_connector, 
      ]
}

resource "google_compute_network_peering_routes_config" "cloud_sql_peering_routes" { 
    import_custom_routes = true
    export_custom_routes = true
    peering = google_service_networking_connection.cloud_sql_private_vpc_connection.peering
    network = google_compute_network.cloud_sql_private_network.name
}

# --- Cloud SQL Database ---
resource "google_sql_database" "cloud_sql_database" {
    name = var.db_name
    instance = google_sql_database_instance.cloud_sql_instance.name
    project = var.project_id

    depends_on = [ 
      google_sql_database_instance.cloud_sql_instance,
      google_vpc_access_connector.cloud_sql_connector,
      ]
}

# --- Cloud SQL User ---
resource "google_sql_user" "cloud_sql_user" {
    name = var.db_user
    instance = google_sql_database_instance.cloud_sql_instance.name
    password = random_password.db_password.result
    project = var.project_id

    depends_on = [ google_sql_database_instance.cloud_sql_instance ]
}

# --- Secret Manager ---
resource "google_secret_manager_secret" "cloud_sql_secret" {
   secret_id = var.secret_id
   replication {
     auto {}
   }
   depends_on = [ random_password.db_password ]
}

resource "google_secret_manager_secret_version" "cloud_sql_secret_version" {
  secret = google_secret_manager_secret.cloud_sql_secret.name
  secret_data = random_password.db_password.result
}

resource "google_service_account" "cloud_run_service_account" {
  account_id = "cloud-run-service-account"
  display_name = "Service account for Cloud Run"
}

resource "google_secret_manager_secret_iam_member" "cloud_run_service_account_secret_access" {
  secret_id = google_secret_manager_secret.cloud_sql_secret.id
  role = "roles/secretmanager.secretAccessor"
  member = "serviceAccount:${google_service_account.cloud_run_service_account.email}"
  depends_on = [ google_secret_manager_secret.cloud_sql_secret ]
}

# --- SQL Migrations Firewall Rule ---
data "google_compute_default_service_account" "default" {  
}

resource "google_compute_firewall" "migrations_vm_fw_rule" {
  name    = "private-allow-ssh-to-migrations-vm"
  network = google_compute_network.cloud_sql_private_network.name

  allow {
    protocol = "tcp"
    ports    = ["22"]
  }
  priority = "1100"

  source_ranges = ["0.0.0.0/0"]
  target_service_accounts = ["${data.google_compute_default_service_account.default.email}"]
}
