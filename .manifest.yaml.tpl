image: PUSH_IMAGE:VERSION
manifests:
  - image: PUSH_IMAGE:VERSION-linux-amd64
    platform:
      architecture: amd64
      os: linux
  - image: PUSH_IMAGE:VERSION-windows-1809
    platform:
      architecture: amd64
      os: windows
  - image: PUSH_IMAGE:VERSION-windows-1903
    platform:
      architecture: amd64
      os: windows
  - image: PUSH_IMAGE:VERSION-windows-1909
    platform:
      architecture: amd64
      os: windows
  - image: PUSH_IMAGE:VERSION-windows-2004
    platform:
      architecture: amd64
      os: windows
