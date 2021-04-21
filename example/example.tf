terraform {
  required_providers {
    bash = {
      source = "apparentlymart/bash"
    }
    local = {
      source  = "hashicorp/local"
      version = "~> 2.1.0"
    }
  }
}

data "bash_script" "example" {
  source = file("${path.module}/example.sh.tmpl")
  variables = {
    greeting = "Hello"
    names    = tolist(["Medhi", "Aurynn", "Kat", "Ariel"])
    num      = 3
    ids = tomap({
      a = "i-123"
      b = "i-456"
      c = "i-789"
    })
  }
}

resource "local_file" "example" {
  filename = "${path.module}/example.sh"
  content  = data.bash_script.example.result
}

output "output_filename" {
  value = local_file.example.filename
}
