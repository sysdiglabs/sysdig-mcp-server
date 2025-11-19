{ buildGoModule }:
buildGoModule (finalAttrs: {
  pname = "sysdig-cli-scanner";
  version = "0.4.0";
  src = ./.;
  vendorHash = "sha256-O7dPOQ8BvjF9mlI0A7/g4aOvvUIoQ+ODL2mlwECGMtI=";

  subPackages = [
    "cmd/server"
  ];

  ldflags = [
    "-w"
    "-s"
    "-X main.Version=${finalAttrs.version}"
  ];

  doCheck = false;
  env.CGO_ENABLED = 0;

  meta = {
    description = "Sysdig MCP Server";
    homepage = "https://github.com/sysdiglabs/sysdig-mcp-server";
    mainProgram = "server";
  };
})
