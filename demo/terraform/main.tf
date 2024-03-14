terraform {
    required_providers {
        linode = {
            source = "linode/linode"
            version = ">= 1.0.0"
        }
    }
}

variable "instance_count" {
    description = "Number of instances to create"
    type        = number
}

variable "linode_token" {
    description = "Linode API token"
    type        = string
}

variable "root_pass" {
    description = "Root password for the instances"
    type        = string
}

provider "linode" {
    token = var.linode_token
}

locals {
    regions =  ["ca-central", "us-east", "nl-ams", "eu-west", "es-mad", "ap-south", "in-maa", "ap-southeast", "br-gru"]
}

resource "linode_instance" "nanode" {
    count      = var.instance_count
    image      = "linode/ubuntu20.04"
    label      = "nanode-${count.index}"
    region     = element(local.regions, count.index % length(local.regions))
    type       = "g6-nanode-1"
    root_pass  = var.root_pass
    authorized_keys = ["${trimspace(file("../ssh-keys/demo-ssh-keys.pub"))}"]

    provisioner "remote-exec" {
        connection {
            type        = "ssh"
            user        = "root"
            private_key = file("../ssh-keys/demo-ssh-keys")
            host     = self.ip_address
            timeout = "30m"
        }
        inline = [
            "sudo apt-get update",
            "sudo apt-get install -y python3 python3-pip",
            "pip3 install ansible",
            "sudo apt install -y git",
        ]
    }
}

output "instance_details" {
    value = { for instance in linode_instance.nanode : instance.label => instance.ip_address }
    description = "Details of Linode instances (name and IP address)"
}
