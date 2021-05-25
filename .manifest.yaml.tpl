image: PUSH_IMAGE:VERSION
manifests:
  - image: PUSH_IMAGE:VERSION-windows-1809
    platform:
      architecture: amd64
      os: windows
  - image: PUSH_IMAGE:VERSION-windows-2004
    platform:
      architecture: amd64
      os: windows
  - image: PUSH_IMAGE:VERSION-windows-20H2
    platform:
      architecture: amd64
      os: windows
