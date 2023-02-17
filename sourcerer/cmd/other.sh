function heyo() {
  echo "$1"
  echo "${@:2}"
}

heyo
echo OUT "$1"
echo OUT "${@:2}"
