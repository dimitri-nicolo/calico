# anomaly-detection-api

REST API supporting Anomaly-Detection. Currently supporting storage of models created through training cycles.

## Endpoints

### `/clusters/{cluster-name}/models/...`

Endpoints to save, load, and stat model files called by Anomaly-Detection pods.

#### POST /clusters/{cluser_name}/models/dynamic/{detector_category}/{detector_class} 

Saves a model provided in the body of the request. If a model already exists for the specified  /clusters/{cluser_name}/models/dynamic/{detector_category}/{detector_class} path, it will replace the existing model with the one provided in the body.  The cluster name and detector name will be validated that they are RFC1123 compliant, and the file size in the body to be capped at 100 Mb.

##### Request Fields

| Name              | Type   | In     | Description                                               |
|-------------------|--------|--------|-----------------------------------------------------------|
| content-type      | string | header | text/plain indicates base64 string of binary body content |
| authorization     | string | header | Bearer “<service-account-token>”                          |
| cluster_name      | string | path   | Specifies which cluster the model is for                  |
| detector_name     | string | path   | Specifies which detector the model is for                 |
| detector_category | string | path   | Sub category of a model                                   |
| body              | string | body   | Base64 string of model file content                       |

##### Response Codes

| Name | Type                                                     |
|------|----------------------------------------------------------|
| 200  | OK                                                       |
| 401  | If empty or malformed token                              |
| 403  | If token is matched but does not have the correct roles  |
| 413  | When requests' suze exceeds the server's file size limit |
| 400  | Validation with cluster and detector names failed        |
| 500  | For any server error                                     |

#### GET /clusters/{cluser_name}/models/dynamic/{detector_category}/{detector_class} 

This endpoint is used to serve downloading an existing model for the specified cluster and model.  It returns the model as a base64 string of the model's byte content. 

##### Request Fields

| Name              | Type   | In     | Description                                               |
|-------------------|--------|--------|-----------------------------------------------------------|
| authorization     | string | header | Bearer “<service-account-token>”                          |
| cluster_name      | string | path   | Specifies which cluster the model is for                  |
| detector_name     | string | path   | Specifies which detector the model is for                 |
| detector_category | string | path   | Sub category of a model                                   |

##### Response Codes

| Name | Type                                                     |
|------|----------------------------------------------------------|
| 200  | OK                                                       |
| 401  | If empty or malformed token                              |
| 403  | If token is matched but does not have the correct roles  |
| 404  | Model file on request path does not exist                |
| 400  | Validation with cluster and detector names failed        |
| 500  | For any server error                                     |

#### HEAD /clusters/{cluser_name}/models/dynamic/{detector_category}/{detector_class} 

This endpoint is used to serve as a file information endpoint, intended to be used as a pre-flight check before downloading a model - GET /clusters/…  The expected size of the model (before base64 encoding) is added as the response header Content-Length

##### Request Fields

| Name              | Type   | In     | Description                                               |
|-------------------|--------|--------|-----------------------------------------------------------|
| authorization     | string | header | Bearer “<service-account-token>”                          |
| cluster_name      | string | path   | Specifies which cluster the model is for                  |
| detector_name     | string | path   | Specifies which detector the model is for                 |
| detector_category | string | path   | Sub category of a model                                   |

##### Response Codes

| Name | Type                                                     |
|------|----------------------------------------------------------|
| 200  | OK                                                       |
| 401  | If empty or malformed token                              |
| 403  | If token is matched but does not have the correct roles  |
| 404  | Model file on request path does not exist                |
| 400  | Validation with cluster and detector names failed        |
| 500  | For any server error                                     |