
function _sourcerer_initializer {
  Push-Location
  Set-Location (Split-Path $PSCommandPath)
  $Local:tmpOut = New-TemporaryFile
  go run . source sourcerer > $Local:tmpOut
  Copy-Item "$Local:tmpOut" "$Local:tmpOut.ps1"
  . "$Local:tmpOut.ps1"
  Pop-Location
}
. _sourcerer_initializer

Set-Alias mc mancli
