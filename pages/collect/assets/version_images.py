import socket
import json

def get_containers():
    # Create a Unix socket
    sock = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)

    # Connect the socket to the Docker daemon
    sock.connect("/var/run/docker.sock")

    # Send HTTP request to Docker API
    request = b"GET /containers/json HTTP/1.0\r\n\r\n"
    sock.sendall(request)

    # Receive the HTTP response from Docker API
    response = []
    while True:
        data = sock.recv(4096)
        if not data:
            break
        response.append(data)

    # Close the Unix socket
    sock.close()

    # Parse the HTTP response
    raw_response = b"".join(response).split(b"\r\n\r\n", 1)[1]
    containers = json.loads(raw_response)

    return containers

containers = get_containers()
json_output = []
for container in containers:
    image_info = container["Image"].split(":")
    if len(image_info) == 1:
        image_info.append("latest")
    container_names = ", ".join(name.lstrip("/") for name in container["Names"])
    json_output.append({"Image": image_info[0], "Tag": image_info[1], "Name": container_names})

# Use json.dumps to print JSON output
print(json.dumps(json_output))