###-begin-pnpm-completion-###
_pnpm_completion() {
  local words cword completion
  if type _get_comp_words_by_ref &>/dev/null; then
    _get_comp_words_by_ref -n = -n @ -n : -w words -i cword
  else
    cword="$COMP_CWORD"
    words=("${COMP_WORDS[@]}")
  fi
  COMPREPLY=()
  while IFS= read -r completion; do
    printf -v completion '%q' "$completion"
    COMPREPLY+=("$completion")
  done < <(COMP_CWORD="$cword" COMP_LINE="$COMP_LINE" COMP_POINT="$COMP_POINT" SHELL=bash pnpm completion-server -- "${words[@]}")
  if type __ltrim_colon_completions &>/dev/null; then
    __ltrim_colon_completions "${words[cword]}"
  fi
}
complete -F _pnpm_completion pnpm pn
###-end-pnpm-completion-###
