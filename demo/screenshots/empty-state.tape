Set Shell "bash"
Set Width 1000
Set Height 650
Set FontSize 16
Set Padding 12
Set Theme "Dracula"
Set WindowBar "Colorful"
Set WindowBarSize 32
Set BorderRadius 8
Set TypingSpeed 20ms

Hide
Type "unset NO_COLOR; export TERM=xterm-256color COLORTERM=truecolor CLICOLOR_FORCE=1; clear"
Enter
Show
Type "./bin/setup-linux-amd64 --demo"
Enter
Wait+Screen /Setup Admin Console/
Enter
Wait+Screen /Fresh Setup/
Enter
Wait+Screen /select at least one fresh setup action/
Sleep 500ms
Screenshot "docs/assets/screenshots/empty-state.png"
Sleep 500ms
Ctrl+C
Sleep 300ms
