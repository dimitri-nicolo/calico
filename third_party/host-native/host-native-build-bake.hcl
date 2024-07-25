group "default" {
  targets = ["rhel"]
}

group "rhel" {
  targets = ["rhel8", "rhel9"]
}

target "rhel8" {
  dockerfile = "Dockerfile.rhel8"
  tags = ["calico/host-native-build:rhel8"]
}

target "rhel9" {
  dockerfile = "Dockerfile.rhel9"
  tags = ["calico/host-native-build:rhel9"]
}
