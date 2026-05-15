from http.server import BaseHTTPRequestHandler, HTTPServer
import json
import time
from urllib.parse import urlparse


USERS = [
    {"id": "1", "email": "john@example.com", "displayName": "John"},
    {"id": "2", "email": "jane@example.com", "displayName": "Jane"},
]

PROFILES = []


class Handler(BaseHTTPRequestHandler):
    def _send_json(self, status_code, payload):
        body = json.dumps(payload).encode("utf-8")

        self.send_response(status_code)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def _read_json_body(self):
        content_length = int(self.headers.get("Content-Length", "0"))
        body = self.rfile.read(content_length)

        try:
            return json.loads(body.decode("utf-8")), None
        except json.JSONDecodeError:
            return None, {"error": "invalid json"}

    def _send_empty_json(self, status_code):
        self.send_response(status_code)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", "0")
        self.end_headers()

    def _send_body_not_allowed_test(self, status_code):
        body = json.dumps({"deleted": True}).encode("utf-8")

        self.send_response(status_code)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def _send_invalid_json(self, status_code):
        body = b'{status: ok}'

        self.send_response(status_code)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def do_GET(self):
        path = urlparse(self.path).path

        if path == "/health":
            self._send_json(200, {})
            return

        if path == "/users":
            self._send_json(200, USERS)
            return

        if path == "/profiles":
            self._send_json(200, PROFILES)
            return

        if path.startswith("/users/"):
            user_id = path.removeprefix("/users/")

            for user in USERS:
                if user["id"] == user_id:
                    self._send_json(200, user)
                    return

            self._send_json(404, {"error": "user not found"})
            return

        if path == "/timeout":
            time.sleep(10)
            self._send_json(200, {"message": "slow response completed"})
            return

        if path == "/reset":
            self.close_connection = True
            return

        self._send_json(404, {"error": "route not found"})

    def do_POST(self):
        path = urlparse(self.path).path

        if path == "/users":
            payload, err = self._read_json_body()
            if err:
                self._send_json(400, err)
                return

            user = {
                "id": str(len(USERS) + 1),
                "email": payload.get("email"),
                "displayName": payload.get("displayName"),
            }

            USERS.append(user)
            self._send_json(201, user)
            return

        if path == "/profiles":
            payload, err = self._read_json_body()
            if err:
                self._send_json(400, err)
                return

            profile = {
                "id": str(len(PROFILES) + 1),
                "userId": payload.get("userId"),
                "contact": payload.get("contact"),
                "preferences": payload.get("preferences"),
            }

            PROFILES.append(profile)
            self._send_json(201, profile)
            return

        self._send_json(404, {"error": "route not found"})

    def do_DELETE(self):
        path = urlparse(self.path).path

        if path.startswith("/users/"):
            user_id = path.removeprefix("/users/")

            for user in USERS:
                if user["id"] == user_id:
                    USERS.remove(user)

                    # Intentionally wrong for testing:
                    # OpenAPI says 204 has no response content,
                    # but this sends a JSON body.
                    self._send_body_not_allowed_test(204)
                    return

            self._send_json(404, {"error": "user not found"})
            return

        self._send_json(404, {"error": "route not found"})


def main():
    server = HTTPServer(("localhost", 8081), Handler)
    print("Test service listening on http://localhost:8081")
    server.serve_forever()


if __name__ == "__main__":
    main()
