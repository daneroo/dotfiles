###-begin-pnpm-completion-###
_pnpm_completion() {
  local line="$COMP_LINE" word completion replaced_prefix=""
  local -a words=()
  local i
  for ((i = 0; i <= COMP_CWORD; i++)); do
    word="${COMP_WORDS[i]}"
    if ((${#words[@]} > 0)) && [[ -n "$word" && "$line" != [[:space:]]* ]]; then
      words[${#words[@]}-1]+="$word"
    else
      line="${line#"${line%%[![:space:]]*}"}"
      words+=("$word")
    fi
    line="${line#"$word"}"
  done
  word="${words[${#words[@]}-1]}"
  for ((i = ${#word} - 1; i >= 0; i--)); do
    if [[ "$COMP_WORDBREAKS" == *"${word:i:1}"* ]]; then
      replaced_prefix="${word:0:i+1}"
      break
    fi
  done
  COMPREPLY=()
  while IFS= read -r completion; do
    printf -v completion '%q' "${completion#"$replaced_prefix"}"
    COMPREPLY+=("$completion")
  done < <(COMP_CWORD="$((${#words[@]} - 1))" COMP_LINE="$COMP_LINE" COMP_POINT="$COMP_POINT" SHELL=bash pnpm completion-server -- "${words[@]}")
}
complete -F _pnpm_completion pnpm pn
###-end-pnpm-completion-###
