{ buildGoModule, versionCheckHook }:
buildGoModule (finalAttrs: {
  pname = "sysdig-mcp-server";
  version = "1.0.0";
  src = ./.;
  # This hash is automatically re-calculated with `just rehash-package-nix`. This is automatically called as well by `just update`.
  vendorHash = "sha256-qMgFlDqzmtpxNOFCX9TsE4sjz0ZdvTJ5Q5IpA8lzG8g=";

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

  postInstall = ''
    mv $out/bin/server $out/bin/sysdig-mcp-server
  '';

  nativeInstallCheckInputs = [ versionCheckHook ];
  doInstallCheck = true;

  meta = {
    description = "Sysdig MCP Server";
    homepage = "https://github.com/sysdiglabs/sysdig-mcp-server";
    mainProgram = "sysdig-mcp-server";
  };
})
