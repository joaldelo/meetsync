# MeetSync API Documentation

This directory contains the OpenAPI documentation for the MeetSync API.

## OpenAPI Specification

The OpenAPI specification is defined in the `openapi.yaml` file. This file describes all the endpoints, request/response schemas, and other details of the API.

## Viewing the Documentation

You can view the documentation in several ways:

1. **Swagger UI**: You can use the Swagger UI to view and interact with the API documentation. There are several online tools available, such as:
   - [Swagger Editor](https://editor.swagger.io/)
   - [Redocly](https://redocly.github.io/redoc/)

2. **Local Swagger UI**: You can run a local Swagger UI instance using Docker:
   ```bash
   docker run -p 8081:8080 -e SWAGGER_JSON=/docs/openapi.yaml -v $(pwd)/docs:/docs swaggerapi/swagger-ui
   ```
   Then open your browser to http://localhost:8081

## Generating Client Libraries

You can use the OpenAPI specification to generate client libraries for various programming languages. Tools like [OpenAPI Generator](https://openapi-generator.tech/) can be used for this purpose.

Example:
```bash
# Install OpenAPI Generator
npm install @openapitools/openapi-generator-cli -g

# Generate a TypeScript client
openapi-generator-cli generate -i docs/openapi.yaml -g typescript-fetch -o client/typescript

# Generate a Python client
openapi-generator-cli generate -i docs/openapi.yaml -g python -o client/python
```

## Updating the Documentation

When making changes to the API, please update the OpenAPI specification accordingly. This ensures that the documentation stays in sync with the actual implementation. 