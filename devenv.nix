{ pkgs, lib, config, inputs, ... }:
let
  # Định nghĩa npm packages cần cài toàn cục
  myNpmPackages = with pkgs.nodePackages; [
    "@google/gemini-cli"   # gemini-cli
    typescript             # ví dụ thêm
  ];

in
{
  name = "SeaweedFS";
  # https://devenv.sh/basics/
  env.GREET = "devenv";

  # https://devenv.sh/packages/
  packages = [ pkgs.git ];

  languages.go.enable = true;

  # https://devenv.sh/scripts/
  scripts.hello.exec = ''
    echo hello from $GREET
  '';

  scripts.devenv-init.exec = ''
    curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.40.3/install.sh | bash
    bash ~/.nvm/nvm.sh
    nvm install 24
    npm install -g @google/gemini-cli
    gemini --version
    npm i -g @openai/codex
    codex --version
    npm i -g opencode-web@latest
    npm install -g @qwen-code/qwen-code@latest
    qwen --version
  '';


  # https://devenv.sh/basics/
  enterShell = ''
    hello         # Run scripts directly
    git --version # Use packages
  '';

  # https://devenv.sh/tasks/
  # tasks = {
  #   "myproj:setup".exec = "mytool build";
  #   "devenv:enterShell".after = [ "myproj:setup" ];
  # };

  # https://devenv.sh/tests/
  enterTest = ''
    echo "Running tests"
    git --version | grep --color=auto "${pkgs.git.version}"
  '';

  # https://devenv.sh/git-hooks/
  # git-hooks.hooks.shellcheck.enable = true;

}
