{ buildGoModule }:
buildGoModule (finalAttrs: {
  pname = "sysdig-cli-scanner";
  version = "0.4.0";
  src = ./.;
  vendorHash = "sha256-hZd6N3lPSLky7sJfD6zaj0xBr8irXRh2ckE/Lp9BmH4=";

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
