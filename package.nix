{ buildGo126Module, versionCheckHook }:
buildGo126Module (finalAttrs: {
  pname = "sysdig-mcp-server";
  version = "1.0.7";
  src = ./.;
  # This hash is automatically re-calculated with `just rehash-package-nix`. This is automatically called as well by `just update`.
  vendorHash = "sha256-OtXl71IUEq+n+tL9q79t2qq68uwj4a4MLJBGCvZwy0o=";

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
