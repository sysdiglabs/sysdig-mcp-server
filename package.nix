{ buildGo126Module, versionCheckHook }:
buildGo126Module (finalAttrs: {
  pname = "sysdig-mcp-server";
  version = "2.0.0";
  src = ./.;
  # This hash is automatically re-calculated with `just rehash-package-nix`. This is automatically called as well by `just update`.
  vendorHash = "sha256-diD5m9+O7Qn7ln7F/R22Iaz2wNf7yth7ef4JEafrR9o=";

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
