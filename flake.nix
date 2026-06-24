{
  description = "the-run — runner race results platform";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

  outputs =
    { self, nixpkgs }:
    let
      system = "x86_64-linux";
      pkgs = import nixpkgs { inherit system; };
    in
    {
      devShells.${system}.default = pkgs.mkShell {
        packages = with pkgs; [
          go
          gopls
          gotools
          golangci-lint

          nodejs_22
          pnpm

          pulumi
          pulumiPackages.pulumi-go

          awscli2

          just
          zip
          jq
          curl
        ];

        shellHook = ''
          export GOFLAGS="-mod=mod"
          echo "the-run dev shell — go $(go version | awk '{print $3}'), node $(node --version), pulumi $(pulumi version)"
        '';
      };
    };
}
