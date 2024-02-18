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
    regions =  ["us-east", "nl-ams", "eu-west", "ap-south"]
}

resource "linode_instance" "nanode" {
    count      = var.instance_count
    image      = "linode/ubuntu20.04"
    label      = "nanode-${count.index}"
    region     = element(local.regions, count.index % length(local.regions))
    type       = "g6-nanode-1"
    root_pass  = var.root_pass
    authorized_keys = ["${trimspace(file("~/.ssh/id_rsa.pub"))}"]

    provisioner "remote-exec" {
        connection {
            type        = "ssh"
            user        = "root"
            private_key = file("~/.ssh/id_rsa")
            host     = self.ip_address
            timeout = "30m"
        }
        inline = [
            "sudo apt-get update",
            "sudo apt-get install -y python3 python3-pip",
            "pip3 install ansible",
            "sudo apt-get remove -y docker docker-engine docker.io",
            "sudo apt-get install -y apt-transport-https ca-certificates curl gnupg lsb-release",
            "curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /usr/share/keyrings/docker-archive-keyring.gpg",
            "echo \"deb [arch=amd64 signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable\" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null",
            "sudo apt-get update -y",
            "sudo apt-get install -y docker-ce docker-ce-cli containerd.io",
            "sudo systemctl start docker",
            "sudo systemctl enable docker",
            "sudo systemctl enable containerd",
            "sudo apt install -y git",
            "wget https://go.dev/dl/go1.22.0.linux-amd64.tar.gz",
            "rm -rf /usr/local/go && tar -C /usr/local -xzf go1.22.0.linux-amd64.tar.gz",
            "export PATH=$PATH:/usr/local/go/bin",
            "git clone https://github.com/pierreleocadie/SecuraChain.git"
        ]
    }
}