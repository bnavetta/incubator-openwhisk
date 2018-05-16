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

  count = 3
}

resource "aws_ebs_volume" "registry_data" {
  availability_zone = "${aws_instance.worker.0.availability_zone}"
  size              = 40
}

resource "aws_volume_attachment" "registry_attachment" {
  device_name = "/dev/sdh"
  volume_id   = "${aws_ebs_volume.registry_data.id}"
  instance_id = "${aws_instance.worker.0.id}"
}

resource "aws_security_group" "openwhisk" {
  name        = "openwhisk"
  description = "Security group for OpenWhisk"

  ingress {
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  # HTTP for the controller
  ingress {
    from_port   = 80
    to_port     = 80
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  # For remote docker
  ingress {
    from_port   = 4243
    to_port     = 4243
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  # For the Docker registry
  ingress {
    from_port   = 5000
    to_port     = 5000
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  # For CouchDB
  ingress {
    from_port   = 5984
    to_port     = 5984
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  # For ZooKeeper
  ingress {
    from_port   = 2180
    to_port     = 2190
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  # For Kafka
  ingress {
    from_port = 9072
    to_port   = 9072
    protocol  = "tcp"
  }

  ingress {
    from_port   = 9090
    to_port     = 9100
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  ingress {
    from_port   = 8090
    to_port     = 8100
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  # Etcd
  ingress {
    from_port   = 2379
    to_port     = 2379
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  ingress {
    from_port   = 0
    to_port     = 20000
    protocol    = "tcp"
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
