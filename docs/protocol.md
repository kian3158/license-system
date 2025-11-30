# Protocol: register / report / heartbeat

POST /register
Request:
{
  "client_id": "uuid",
  "pub_key": "base64",
  "fingerprint": {"machine_id": "..."},
  "app_id": "string",
  "version": "string"
}
Response:
{
  "license": { ... },
  "signature": "base64",
  "server_pub_key": "base64"
}

POST /report
Request:
{
  "client_id":"uuid",
  "license_id":"LIC-...",
  "timestamp":"ISO-8601",
  "usage_bytes_since_last": 1234,
  "total_usage_bytes": 12345,
  "signature":"base64"
}
Response:
{ "status":"ok", "allowed": true/false, "remaining_bytes": 123 }

GET /heartbeat
Response:
{ "status":"ok", "server_time":"ISO-8601" }
