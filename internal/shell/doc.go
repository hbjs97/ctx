// Package shell provides shell integration for automatic profile activation.
// It generates shell hook snippets (chpwd for Zsh, PROMPT_COMMAND for Bash,
// --on-variable for Fish) that call ctx activate on directory change.
package shell
