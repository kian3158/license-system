from client import LicenseClient

if __name__ == "__main__":
    client = LicenseClient()
    client.ping()
    client.heartbeat()
    client.register()
    client.report(10 * 1024 * 1024)
    client.report(110 * 1024 * 1024)
