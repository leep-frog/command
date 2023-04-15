
function _sourcerer_initializer {
  Push-Location
  Set-Location (Split-Path $PSCommandPath)
  $Local:tmpOut = New-TemporaryFile
  go run . source sourcerer > $Local:tmpOut
  . $Local:tmpOut
  Pop-Location
}
_sourcerer_initializer
