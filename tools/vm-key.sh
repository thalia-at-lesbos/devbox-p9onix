#!/bin/bash
# send keystrokes to the 9front VM via virsh send-key
V="virsh -c qemu:///session"
D=9front
key(){ $V send-key $D --codeset linux "$@" >/dev/null; }
sh(){ key KEY_LEFTSHIFT "$1"; }
typestr(){
  local s="$1" c
  for (( i=0; i<${#s}; i++ )); do
    c="${s:i:1}"
    case "$c" in
      [a-z]) key "KEY_$(printf %s "$c" | tr a-z A-Z)" ;;
      [A-Z]) sh "KEY_$c" ;;
      [0-9]) key "KEY_$c" ;;
      ' ')  key KEY_SPACE ;;
      '/')  key KEY_SLASH ;;
      '.')  key KEY_DOT ;;
      ',')  key KEY_COMMA ;;
      '-')  key KEY_MINUS ;;
      '=')  key KEY_EQUAL ;;
      ';')  key KEY_SEMICOLON ;;
      "'")  key KEY_APOSTROPHE ;;
      '\')  key KEY_BACKSLASH ;;
      '[')  key KEY_LEFTBRACE ;;
      ']')  key KEY_RIGHTBRACE ;;
      '`')  key KEY_GRAVE ;;
      '_')  sh KEY_MINUS ;;
      '+')  sh KEY_EQUAL ;;
      ':')  sh KEY_SEMICOLON ;;
      '"')  sh KEY_APOSTROPHE ;;
      '|')  sh KEY_BACKSLASH ;;
      '{')  sh KEY_LEFTBRACE ;;
      '}')  sh KEY_RIGHTBRACE ;;
      '~')  sh KEY_GRAVE ;;
      '?')  sh KEY_SLASH ;;
      '<')  sh KEY_COMMA ;;
      '>')  sh KEY_DOT ;;
      '!')  sh KEY_1 ;;
      '@')  sh KEY_2 ;;
      '#')  sh KEY_3 ;;
      '$')  sh KEY_4 ;;
      '%')  sh KEY_5 ;;
      '^')  sh KEY_6 ;;
      '&')  sh KEY_7 ;;
      '*')  sh KEY_8 ;;
      '(')  sh KEY_9 ;;
      ')')  sh KEY_0 ;;
      *) echo "unmapped char: $c" >&2; return 1 ;;
    esac
  done
}
case "$1" in
  type)  typestr "$2" ;;
  enter) key KEY_ENTER ;;
  line)  typestr "$2" && key KEY_ENTER ;;
  raw)   shift; key "$@" ;;
  *) echo "usage: k.sh {type|line|enter|raw}" >&2; exit 2 ;;
esac
