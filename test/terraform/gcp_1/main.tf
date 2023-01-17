resource "google_compute_network" "vpc_network" {
  name                    = "cbf-network"
  auto_create_subnetworks = false
  mtu                     = 1460
}

resource "google_compute_subnetwork" "default" {
  name          = "cbf-subnet"
  ip_cidr_range = "10.0.1.0/24"
  region        = "europe-west9"
  network       = google_compute_network.vpc_network.id
}

resource "google_compute_instance" "default" {
  name         = "cbf-test-vm"
  machine_type = "custom-1-2480"
  zone         = "europe-west9-a"
  tags         = ["ssh"]

  boot_disk {
    initialize_params {
      image = "debian-cloud/debian-11"
      size = 567
      type = "pd-balanced"
    }
  }

  scratch_disk {
    interface = "NVME"
  }
  scratch_disk {
    interface = "NVME"
  }

  # Install Flask
  metadata_startup_script = "sudo apt-get update; sudo apt-get install -yq build-essential python3-pip rsync; pip install flask"

  network_interface {
    subnetwork = google_compute_subnetwork.default.id

    access_config {
      # Include this section to give the VM an external IP address
    }
  }
}

resource "google_compute_instance" "foo" {
  name         = "cbf-test-other"
  machine_type = "custom-2-4098"
  min_cpu_platform = "Intel Cascade Lake"
  zone         = "europe-west9-a"
  tags         = ["ssh"]

  boot_disk {
    initialize_params {
      image = "debian-cloud/debian-11"
    }
  }

  # Install Flask
  metadata_startup_script = "sudo apt-get update; sudo apt-get install -yq build-essential python3-pip rsync; pip install flask"

  network_interface {
    subnetwork = google_compute_subnetwork.default.id

    access_config {
      # Include this section to give the VM an external IP address
    }
  }
}