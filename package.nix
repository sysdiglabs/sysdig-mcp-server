{ buildGoModule, versionCheckHook }:
buildGoModule (finalAttrs: {
  pname = "sysdig-mcp-server";
  version = "0.4.0";
  src = ./.;
  vendorHash = "sha256-jf/px0p88XbfuSPMry/qZcfR0QPTF9IrPegg2CwAd6M=";

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
