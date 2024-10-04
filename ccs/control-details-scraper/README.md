## CCS Control Details Scraper

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
This json list is used by the CCS Control Details Indexer to be inserted into Elasticsearch.