// Package docs Code generated by swaggo/swag. DO NOT EDIT
package docs

import "github.com/swaggo/swag"

const docTemplate = `{
    "schemes": {{ marshal .Schemes }},
    "swagger": "2.0",
    "info": {
        "description": "{{escape .Description}}",
        "title": "{{.Title}}",
        "contact": {},
        "version": "{{.Version}}"
    },
    "host": "{{.Host}}",
    "basePath": "{{.BasePath}}",
    "paths": {
        "/": {
            "get": {
                "description": "Get a list of all commands processed by the system",
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "Getting commands"
                ],
                "summary": "Retrieve all commands",
                "responses": {
                    "200": {
                        "description": "List of commands",
                        "schema": {
                            "type": "array",
                            "items": {
                                "$ref": "#/definitions/models.Command"
                            }
                        }
                    },
                    "500": {
                        "description": "Server error",
                        "schema": {
                            "$ref": "#/definitions/models.Error"
                        }
                    }
                }
            },
            "post": {
                "description": "Add a new non-sudo command to the system",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "Commands creating"
                ],
                "summary": "Create a new command",
                "parameters": [
                    {
                        "description": "Create command",
                        "name": "command",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "type": "string"
                        }
                    }
                ],
                "responses": {
                    "202": {
                        "description": "Command is being queued",
                        "schema": {
                            "$ref": "#/definitions/models.Message"
                        }
                    },
                    "400": {
                        "description": "Error response",
                        "schema": {
                            "$ref": "#/definitions/models.Error"
                        }
                    },
                    "500": {
                        "description": "Error response on server side",
                        "schema": {
                            "$ref": "#/definitions/models.Error"
                        }
                    }
                }
            }
        },
        "/commands/{id}/fstart": {
            "post": {
                "description": "Forcefully start a queued command by its ID, bypassing queue constraints",
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "Fetching commands"
                ],
                "summary": "Force start a command",
                "parameters": [
                    {
                        "type": "integer",
                        "description": "Command ID",
                        "name": "id",
                        "in": "path",
                        "required": true
                    }
                ],
                "responses": {
                    "200": {
                        "description": "Command started successfully",
                        "schema": {
                            "$ref": "#/definitions/models.Message"
                        }
                    },
                    "400": {
                        "description": "Invalid ID supplied",
                        "schema": {
                            "$ref": "#/definitions/models.Error"
                        }
                    },
                    "404": {
                        "description": "Command not found",
                        "schema": {
                            "$ref": "#/definitions/models.Error"
                        }
                    },
                    "500": {
                        "description": "Server error",
                        "schema": {
                            "$ref": "#/definitions/models.Error"
                        }
                    }
                }
            }
        },
        "/queue": {
            "get": {
                "description": "Get a list of all commands currently in the queue",
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "Queue"
                ],
                "summary": "Retrieve command queue",
                "responses": {
                    "200": {
                        "description": "List of queued items",
                        "schema": {
                            "type": "array",
                            "items": {
                                "$ref": "#/definitions/models.Queue"
                            }
                        }
                    },
                    "500": {
                        "description": "Server error",
                        "schema": {
                            "$ref": "#/definitions/models.Error"
                        }
                    }
                }
            }
        },
        "/sudo": {
            "post": {
                "description": "Add a new sudo command to the system",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "Commands creating"
                ],
                "summary": "Create a new sudo command",
                "parameters": [
                    {
                        "description": "Create sudo command",
                        "name": "command",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "type": "string"
                        }
                    }
                ],
                "responses": {
                    "202": {
                        "description": "Command is being queued",
                        "schema": {
                            "$ref": "#/definitions/models.Message"
                        }
                    },
                    "400": {
                        "description": "Error response",
                        "schema": {
                            "$ref": "#/definitions/models.Error"
                        }
                    },
                    "500": {
                        "description": "Error response on server side",
                        "schema": {
                            "$ref": "#/definitions/models.Error"
                        }
                    }
                }
            }
        },
        "/{id}": {
            "get": {
                "description": "Retrieve a specific command by its unique ID",
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "Getting commands"
                ],
                "summary": "Get a command by ID",
                "parameters": [
                    {
                        "type": "integer",
                        "description": "Command ID",
                        "name": "id",
                        "in": "path",
                        "required": true
                    }
                ],
                "responses": {
                    "200": {
                        "description": "Command detail",
                        "schema": {
                            "$ref": "#/definitions/models.Command"
                        }
                    },
                    "400": {
                        "description": "Invalid ID supplied",
                        "schema": {
                            "$ref": "#/definitions/models.Error"
                        }
                    },
                    "404": {
                        "description": "Command not found",
                        "schema": {
                            "$ref": "#/definitions/models.Error"
                        }
                    },
                    "500": {
                        "description": "Problem on server side",
                        "schema": {
                            "$ref": "#/definitions/models.Error"
                        }
                    }
                }
            }
        },
        "/{id}/stop": {
            "post": {
                "description": "Stop a running command by its ID",
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "Fetching commands"
                ],
                "summary": "Stop a command",
                "parameters": [
                    {
                        "type": "integer",
                        "description": "Command ID",
                        "name": "id",
                        "in": "path",
                        "required": true
                    }
                ],
                "responses": {
                    "200": {
                        "description": "Command stopped successfully",
                        "schema": {
                            "$ref": "#/definitions/models.Message"
                        }
                    },
                    "400": {
                        "description": "Invalid ID supplied",
                        "schema": {
                            "$ref": "#/definitions/models.Error"
                        }
                    },
                    "404": {
                        "description": "Command not found",
                        "schema": {
                            "$ref": "#/definitions/models.Error"
                        }
                    },
                    "500": {
                        "description": "Problem on server side",
                        "schema": {
                            "$ref": "#/definitions/models.Error"
                        }
                    }
                }
            }
        }
    },
    "definitions": {
        "models.Command": {
            "type": "object",
            "properties": {
                "createdAt": {
                    "type": "string"
                },
                "id": {
                    "type": "integer"
                },
                "output": {
                    "type": "string"
                },
                "pid": {
                    "type": "integer"
                },
                "script": {
                    "type": "string"
                },
                "status": {
                    "type": "string"
                },
                "updatedAt": {
                    "type": "string"
                }
            }
        },
        "models.Error": {
            "type": "object",
            "properties": {
                "error": {
                    "type": "string"
                }
            }
        },
        "models.Message": {
            "type": "object",
            "properties": {
                "id": {
                    "type": "integer"
                },
                "message": {
                    "type": "string"
                }
            }
        },
        "models.Queue": {
            "type": "object",
            "properties": {
                "commandId": {
                    "type": "integer"
                },
                "queueId": {
                    "type": "integer"
                },
                "status": {
                    "type": "string"
                }
            }
        }
    }
}`

// SwaggerInfo holds exported Swagger Info so clients can modify it
var SwaggerInfo = &swag.Spec{
	Version:          "1.0",
	Host:             "",
	BasePath:         "/api/commands",
	Schemes:          []string{},
	Title:            "BashAPi service",
	Description:      "RestAPI for executing bash commands in Docker with a queue system.",
	InfoInstanceName: "swagger",
	SwaggerTemplate:  docTemplate,
	LeftDelim:        "{{",
	RightDelim:       "}}",
}

func init() {
	swag.Register(SwaggerInfo.InstanceName(), SwaggerInfo)
}
