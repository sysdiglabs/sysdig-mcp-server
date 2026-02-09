{
  dockerTools,
  sysdig-mcp-server,
  stdenv,
}:
let
  arch = if stdenv.hostPlatform.isAarch64 then "arm64" else "amd64";
  baseImageAmd64 = import ./docker-base-amd64.nix;
  baseImageAarch64 = import ./docker-base-aarch64.nix;
  baseImage = dockerTools.pullImage (
    if stdenv.hostPlatform.isAarch64 then baseImageAarch64 else baseImageAmd64
  );
in
dockerTools.buildLayeredImage {
  name = sysdig-mcp-server.pname;
  tag = "${sysdig-mcp-server.version}-${arch}";
  contents = [ sysdig-mcp-server ];
  fromImage = baseImage;
  config = {
    Entrypoint = [ "${sysdig-mcp-server}/bin/sysdig-mcp-server" ];
    User = "1000";
  };
}
