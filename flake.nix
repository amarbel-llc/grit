{
  description = "MCP for git, wow that's grit";

  inputs = {
    nixpkgs-master.url = "github:NixOS/nixpkgs/b28c4999ed71543e71552ccfd0d7e68c581ba7e9";
    nixpkgs.url = "github:NixOS/nixpkgs/23d72dabcb3b12469f57b37170fcbc1789bd7457";
    utils.url = "https://flakehub.com/f/numtide/flake-utils/0.1.102";
    go.url = "github:friedenberg/eng?dir=devenvs/go";
    shell.url = "github:friedenberg/eng?dir=devenvs/shell";
  };

  outputs =
    {
      self,
      nixpkgs,
      utils,
      go,
      shell,
      nixpkgs-master,
    }:
    utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = import nixpkgs {
          inherit system;
          overlays = [
            go.overlays.default
          ];
        };

        version = "0.1.0";

        grit = pkgs.buildGoApplication {
          pname = "grit";
          inherit version;
          src = ./.;
          modules = ./gomod2nix.toml;
          subPackages = [ "cmd/grit" ];

          postInstall = ''
            $out/bin/grit generate-plugin $out/share/purse-first
          '';

          meta = with pkgs.lib; {
            description = "MCP for git, wow that's grit";
            homepage = "https://github.com/friedenberg/grit";
            license = licenses.mit;
          };
        };
      in
      {
        packages = {
          default = grit;
          inherit grit;
        };

        devShells.default = pkgs.mkShell {
          packages = with pkgs; [
            just
          ];

          inputsFrom = [
            go.devShells.${system}.default
            shell.devShells.${system}.default
          ];

          shellHook = ''
            echo "grit - dev environment"
          '';
        };

        apps.default = {
          type = "app";
          program = "${grit}/bin/grit";
        };
      }
    );
}
