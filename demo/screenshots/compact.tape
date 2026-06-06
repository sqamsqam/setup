Set Shell "bash"
Set Width 760
Set Height 840
Set FontSize 16
Set Padding 12
Set Theme "Dracula"
Set WindowBar "Colorful"
Set WindowBarSize 32
Set BorderRadius 8
Set TypingSpeed 12ms

Hide
Type "unset NO_COLOR; export TERM=xterm-256color COLORTERM=truecolor CLICOLOR_FORCE=1; clear"
Enter
Show
Type "./bin/setup-linux-amd64 --demo"
Enter
Wait+Screen /Setup Admin Console/
Enter
Wait+Screen /Fresh Setup/
Space
Enter
Wait+Screen /TIMEZONE/
Enter
Wait+Screen /Review Plan/
Enter
Wait+Screen /Setup actions complete/
Sleep 500ms
Screenshot "docs/assets/screenshots/compact.png"
Sleep 500ms
Ctrl+C
Sleep 300ms
