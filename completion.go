package main

import (
	"fmt"
	"os"
	"strings"
)

func printCompletionUsage() {
	fmt.Print(`Shell completion

Usage:
  oura completion <bash|zsh|fish>

Examples:
  oura completion bash > /etc/bash_completion.d/oura
  oura completion zsh  > ~/.zsh/completions/_oura
  oura completion fish > ~/.config/fish/completions/oura.fish
`)
}

func handleCompletion(args []string) {
	if len(args) != 1 {
		printCompletionUsage()
		os.Exit(1)
	}

	shell := strings.ToLower(args[0])
	switch shell {
	case "bash":
		fmt.Print(bashCompletionScript)
	case "zsh":
		fmt.Print(zshCompletionScript)
	case "fish":
		fmt.Print(fishCompletionScript)
	default:
		printCompletionUsage()
		os.Exit(1)
	}
}

const bashCompletionScript = `# bash completion for oura
_oura_complete() {
  local cur prev words cword
  _init_completion -n : || return

  local commands="auth personal-info personal_info personal today all sleep activity readiness heartrate hrv stress spo2 resilience vo2 workout tag enhanced-tag enhanced_tag session webhook help completion completions json"

  if [[ $cword -eq 1 ]]; then
    COMPREPLY=( $(compgen -W "$commands" -- "$cur") )
    return
  fi

  local cmd=${words[1]}
  case "$cmd" in
    tag|enhanced-tag|enhanced_tag|session)
      local subs="list get"
      if [[ $cword -eq 2 ]]; then
        COMPREPLY=( $(compgen -W "$subs" -- "$cur") )
        return
      fi
      COMPREPLY=( $(compgen -W "--start-date --end-date --next-token --json -j --help -h" -- "$cur") )
      return
      ;;
    personal-info|personal_info|personal)
      COMPREPLY=( $(compgen -W "get --json -j --help -h" -- "$cur") )
      return
      ;;
    webhook)
      local subs="list get create update delete renew types"
      if [[ $cword -eq 2 ]]; then
        COMPREPLY=( $(compgen -W "$subs" -- "$cur") )
        return
      fi
      COMPREPLY=( $(compgen -W "--callback-url --verification-token --event-type --data-type --json -j --help -h" -- "$cur") )
      return
      ;;
    completion|completions)
      COMPREPLY=( $(compgen -W "bash zsh fish" -- "$cur") )
      return
      ;;
  esac
}

complete -F _oura_complete oura
`

const zshCompletionScript = `#compdef oura

_oura() {
  local -a commands
  commands=(
    'auth:Authenticate'
    'personal-info:Personal info'
    'today:Today summary'
    'all:All metrics'
    'sleep:Sleep'
    'activity:Activity'
    'readiness:Readiness'
    'heartrate:Heart rate'
    'hrv:HRV'
    'stress:Stress'
    'spo2:SpO2'
    'resilience:Resilience'
    'vo2:VO2 max'
    'workout:Workouts'
    'tag:Tags'
    'enhanced-tag:Enhanced tags'
    'session:Sessions'
    'webhook:Webhook subscriptions'
    'help:Help'
    'completion:Shell completion'
    'json:Alias for all --json'
  )

  if (( CURRENT == 2 )); then
    _describe -t commands 'oura command' commands
    return
  fi

  local cmd=$words[2]
  case $cmd in
    tag|enhanced-tag|session)
      _values 'subcommand' list get
      _arguments '--start-date[Start date]' '--end-date[End date]' '--next-token[Next token]' '--json[JSON output]' '-j[JSON output]' '--help[Help]' '-h[Help]'
      ;;
    personal-info)
      _values 'subcommand' get
      _arguments '--json[JSON output]' '-j[JSON output]' '--help[Help]' '-h[Help]'
      ;;
    webhook)
      _values 'subcommand' list get create update delete renew types
      _arguments '--callback-url[Callback URL]' '--verification-token[Verification token]' '--event-type[create|update|delete]' '--data-type[Data type]' '--json[JSON output]' '-j[JSON output]' '--help[Help]' '-h[Help]'
      ;;
    completion)
      _values 'shell' bash zsh fish
      ;;
  esac
}

_oura
`

const fishCompletionScript = `# fish completion for oura
complete -c oura -f

set -l cmds auth personal-info today all sleep activity readiness heartrate hrv stress spo2 resilience vo2 workout tag enhanced-tag session webhook help completion json
complete -c oura -n 'test (count (commandline -opc)) -eq 1' -a "$cmds"

# Common flags
complete -c oura -l help -s h -d 'Show help'
complete -c oura -l json -s j -d 'JSON output'

# completion
complete -c oura -n '__fish_seen_subcommand_from completion' -a 'bash zsh fish'

# tag/enhanced-tag/session
for c in tag enhanced-tag session
  complete -c oura -n "__fish_seen_subcommand_from $c" -a 'list get'
  complete -c oura -n "__fish_seen_subcommand_from $c" -l start-date -d 'Start date'
  complete -c oura -n "__fish_seen_subcommand_from $c" -l end-date -d 'End date'
  complete -c oura -n "__fish_seen_subcommand_from $c" -l next-token -d 'Next token'
end

# personal-info
complete -c oura -n '__fish_seen_subcommand_from personal-info' -a 'get'

# webhook
complete -c oura -n '__fish_seen_subcommand_from webhook' -a 'list get create update delete renew types'
complete -c oura -n '__fish_seen_subcommand_from webhook' -l callback-url -d 'Callback URL'
complete -c oura -n '__fish_seen_subcommand_from webhook' -l verification-token -d 'Verification token'
complete -c oura -n '__fish_seen_subcommand_from webhook' -l event-type -d 'create|update|delete'
complete -c oura -n '__fish_seen_subcommand_from webhook' -l data-type -d 'Data type'
`
