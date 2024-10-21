## CCS Control Details Scraper

The CCS Control Details Scraper scrapes information from [regoLibrary](https://github.com/kubescape/regolibrary). 
It then generates a list of controlDetails and writes it into a file to be embedded into the main CCS API application.

This application is run by running: `make gen-controls` in the root directory of the CCS application.

The scraper generates a list of control details that looks like:

```
[{
  "controlDetailsID": "C-0009",
  "name": "C-0009 - Resource limits",
  "description": "Detailed description of the control.",
  "category": "Category Name",
  "subCategory": "SubCategory Name",
  "remediation": "Steps for remediation.",
  "framework": ["Kubescape", "Another Framework"],
  "prerequisites": {
    "cloudProviders": ["AWS", "Azure"]
  },
  "severity": 8.5,
  "frameworkOverrides": [
    {
      "frameworkName": "NIST",
      "name": "Specific Override",
      "description": "Short description.",
      "long_description": "Extended detailed description.",
      "remediation": "Detailed remediation steps.",
      "references": ["Reference 1", "Reference 2"],
      "manualCheck": "Manual check steps",
      "impactStatement": "Impact of not complying",
      "defaultValue": "Default value description"
    }
  ],
  "relatedResources": ["Resource 1", "Resource 2"],
  "testCriteria": "Specific testing criteria.",
  "manualCheck": "Manual check procedures",
  "configuration": [
    {
      "path": "configuration/path",
      "name": "Configuration Name",
      "description": "Description of the configuration"
    }
  ],
  "example": "Example of failure scenario"
}...]
```

The Makefile target above clones the regoLibrary repository, runs the scraper, then deletes the repository.