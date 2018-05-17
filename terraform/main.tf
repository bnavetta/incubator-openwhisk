provider "aws" {
  access_key = "${var.access_key}"
  secret_key = "${var.secret_key}"
  region     = "${var.region}"
}

# Configure a SSH key pair
resource "aws_key_pair" "openwhisk" {
  key_name   = "openwhisk-key"
  public_key = "${var.public_key}"
}

resource "aws_instance" "worker" {
  ami           = "ami-4004883f"
  instance_type = "t2.medium"

  key_name        = "${aws_key_pair.openwhisk.key_name}"
  security_groups = ["${aws_security_group.openwhisk.name}"]

  root_block_device {
    volume_type           = "standard"
    volume_size           = 80
    delete_on_termination = true
  }

  count = 3
}

resource "aws_security_group" "openwhisk" {
  name        = "openwhisk"
  description = "Security group for OpenWhisk"

  # What even are firewalls anyways?

  ingress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
  # Allow all egress
  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}
