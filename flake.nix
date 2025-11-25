{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };
  outputs =
    {
      self,
      nixpkgs,
      flake-utils,
    }:
    let
      overlays.default = final: prev: {
        sysdig-mcp-server = prev.pkgsStatic.callPackage ./package.nix { };
      };
      flake = flake-utils.lib.eachDefaultSystem (
        system:
        let
          pkgs = import nixpkgs {
            inherit system;
            config.allowUnfree = true;
            overlays = [ self.overlays.default ];
          };
        in
        {
          packages = {
            inherit (pkgs) sysdig-mcp-server;
            default = pkgs.sysdig-mcp-server;
          };
          devShells.default =
            with pkgs;
            mkShell {
              packages = [
                ginkgo
                go
                gofumpt
                golangci-lint
                just
                mockgen
                pre-commit
                sd
              ];
              shellHook = ''
                pre-commit install
              '';
            };

          formatter = pkgs.nixfmt-rfc-style;
        }
      );
    in
    flake // { inherit overlays; };
}
