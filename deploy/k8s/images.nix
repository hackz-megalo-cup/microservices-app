let
  imageTag =
    let
      value = builtins.getEnv "IMAGE_TAG";
    in
    if value != "" then value else "latest";

  ghcrImage = name: "ghcr.io/hackz-megalo-cup/${name}:${imageTag}";
in
{
  inherit imageTag ghcrImage;
}
